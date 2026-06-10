# Design: Activeflow-scoped Webhook Subscription

Status: DRAFT (design review in progress)
Author: CPO (Lux) on behalf of pchero
Date: 2026-06-10

## 1. Problem

VoIPbin's outbound webhook subscription unit is **coarse**: a single per-customer
`webhook_uri` (stored in `bin-customer-manager`, cached in `bin-webhook-manager`'s
`pkg/accounthandler`). Every event for that customer fans out to that one URI via
`notifyhandler.PublishWebhookEvent → WebhookV1WebhookSend → SendWebhookToCustomer`.

A customer running many concurrent sessions receives all events on one endpoint and
must filter by `activeflow_id` themselves. There is no way to say "deliver the events
produced by *this* execution instance to *this* endpoint".

## 2. Goal

Allow an Activeflow to carry an **optional, static** webhook destination
(`webhook_uri` + `webhook_method`) set at creation time. Every event whose payload
carries that `activeflow_id` is delivered **additionally** to that destination, on top
of the existing per-customer webhook (non-destructive).

Non-goals (explicitly out of scope for this change):
- HMAC / outbound signing
- Dynamic runtime subscribe/unsubscribe API
- Delivery guarantee / dead-letter / persistent retry rework (inherits current
  fire-and-forget + 3x retry behavior)
- Per-event filtering (all events carrying the activeflow_id are forwarded)

Scope clarification (round 2): the activeflow's OWN lifecycle events
(`activeflow_created/updated/deleted`) serialize the activeflow id as `id`, not
`activeflow_id` (the activeflow model embeds `commonidentity.Identity`). Therefore the
5.2 extractor does NOT treat them as carrying an activeflow_id, and they are NOT
re-delivered to the per-activeflow endpoint. The per-activeflow webhook receives the
DERIVED resource events (call, recording, etc.), not the activeflow's own lifecycle. This
is intended: the activeflow lifecycle is the customer-level concern; the per-activeflow
endpoint is for the session's downstream resources. The phrase "every event whose payload
carries that activeflow_id" in this section is to be read literally (only payloads with an
`activeflow_id` field qualify).

## 3. Confirmed facts (from code)

- `call` and `recording` models already carry `ActiveflowID` and expose it in their
  `WebhookMessage` (`ConvertWebhookMessage`). So the correlation key is already present
  in customer-facing payloads.
- `notifyhandler.PublishWebhookEvent(ctx, customerID, eventType, data)` is the single
  fan-out point used by all services. It does `go PublishEvent(...)` (internal RabbitMQ
  event exchange) **and** `go PublishWebhook(...)` (customer webhook path).
- `bin-flow-manager` already publishes activeflow lifecycle events:
  `activeflow_created` (db.go:117), `activeflow_updated` (db.go:230,254),
  `activeflow_deleted` (db.go:379), all via `PublishWebhookEvent`. These land on the
  `bin-manager.flow-manager.event` exchange.
- `bin-webhook-manager` already runs a `pkg/subscribehandler` (consumes
  `customer-manager` events) and a `pkg/cachehandler` (Redis) + `pkg/accounthandler`
  (caches customer webhook config in Redis).

## 4. Confirmed design decisions (from pchero)

1. **Subscription unit**: activeflow, **static** (uri set at creation, immutable).
2. **Cache**: `bin-webhook-manager` owns a **Redis** cache (extend existing
   `pkg/cachehandler`), populated by subscribing to flow-manager lifecycle events.
3. **Cache miss**: **fallback** to a one-shot RPC to flow-manager to fetch the
   activeflow's webhook config, then backfill the cache. Avoids losing early events
   (e.g. `call_created` arriving before `activeflow_created` is consumed, or after a
   webhook-manager pod restart).
4. **Destination semantics**: **additive** — the activeflow URI is delivered in
   addition to the customer URI, never replacing it.

## 5. Architecture

```
[populate cache]
  bin-webhook-manager subscribes to bin-manager.flow-manager.event
    activeflow_created/updated → if webhook_uri set: Redis SET webhook:activeflow:{id} = {uri,method}, TTL=T_live
                                 else:               Redis SET webhook:activeflow:{id} = {NEGATIVE}, TTL=T_neg
    activeflow_deleted → Redis SET webhook:activeflow:{id} = {NEGATIVE tombstone, tm_delete}, TTL=T_neg

[deliver on hot path]
  Existing event arrives at SendWebhookToCustomer(customerID, dataType, data)
    1. deliver to customer URI (UNCHANGED)
    2. extract activeflow_id from data (if present)
    3. if activeflow_id != nil:
         lookup Redis webhook:activeflow:{id}
           HIT(positive)  → also deliver to that uri/method
           HIT(negative)  → skip (no activeflow webhook configured)
           MISS           → fallback RPC flow-manager.ActiveflowGet(id)
                              → backfill cache (positive or negative)
                              → if positive, deliver
```

### 5.1 Where the activeflow_id lookup lives

The lookup belongs in `bin-webhook-manager` (`pkg/webhookhandler`), NOT in
`bin-common-handler/notifyhandler`. Reasons:
- `notifyhandler` is a shared library bound by the 3+ consumer admission rule; injecting
  a flow-manager dependency there is architecturally wrong.
- The webhook destination decision is webhook-manager's responsibility.

`SendWebhookToCustomer` already receives `data json.RawMessage`. We extract
`activeflow_id` from that JSON (best-effort; absent → skip the extra delivery).

### 5.2 Extracting activeflow_id (CORRECTED after design review round 1)

CRITICAL: the payload that reaches `SendWebhookToCustomer` is NOT the bare resource
`WebhookMessage`. It is wrapped one extra level. Tracing the chain:

- `notifyhandler.PublishWebhook` builds `m := data.CreateWebhookEvent()` (the resource
  WebhookMessage JSON, activeflow_id at top level) — `notifyhandler/publish.go`.
- `requesthandler.WebhookV1WebhookSend` then wraps it again as
  `wmwebhook.Data{Type: eventType, Data: m}` — `requesthandler/webhook_webhooks.go:19-26`.
- `listenhandler.processV1WebhooksPost` marshals `req.Data` (the `webhook.Data` envelope)
  and passes it to `SendWebhookToCustomer` — `listenhandler/v1_webhooks.go:33-39`.

So `data` has the shape `{"type":"call_updated","data":{...activeflow_id here...}}`.
The extraction MUST target the nested `data`:

```go
type webhookEnvelope struct {
    Data struct {
        ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
    } `json:"data"`
}
```

Extracting from the top level (as the original draft proposed) would ALWAYS yield
`uuid.Nil` and the feature would never trigger. Test fixtures in section 10 MUST use the
nested envelope shape, otherwise tests pass against a wrong fixture (false positive).

Only resources that embed `activeflow_id` (call, recording, and future ones) resolve a
non-nil id. Everything else falls through unchanged. No change required in producers.

SECURITY (Option A, final decision): the per-activeflow `webhook_uri` / `webhook_method`
are the customer's OWN data, returned only to the customer's OWN webhook endpoint. They
are NOT a cross-tenant secret. The activeflow lifecycle events
(`activeflow_created` / `activeflow_updated` / `activeflow_deleted`) DO carry
`webhook_uri` / `webhook_method` on `models/activeflow/webhook.go WebhookMessage`. This is
consistent with the customer-level webhook behavior and with the documented
`GET /activeflows` REST response + OpenAPI response schema + RST docs, which all expose
`webhook_uri` / `webhook_method` as response fields. Because `ConvertWebhookMessage()` is
used for BOTH the webhook event payload AND the REST API response, keeping the fields
maintains doc/spec/response consistency.

Consequently webhook-manager pre-populates the cache eagerly from the lifecycle event:
`activeflow_created` / `activeflow_updated` set a POSITIVE entry when `webhook_uri` is
present, or a NEGATIVE entry when it is empty; `activeflow_deleted` writes a NEGATIVE
tombstone carrying `tm_delete`. The fallback path (`FlowV1ActiveflowGet`, full Activeflow
over the internal RPC channel) remains the lazy/miss safety net for events that arrive
before the lifecycle event is consumed (see 5.6).

CUSTOMER GUIDANCE: because the activeflow `webhook_uri` is included in the activeflow
lifecycle webhook payloads delivered to the customer-level webhook endpoint, customers
MUST NOT embed secrets or tokens in the `webhook_uri` query string. This is the customer's
own data delivered to the customer's own endpoint (intended behavior), but a query-string
secret would be visible in those payloads and logs.

### 5.3 Activeflow webhook lifecycle (positive/negative cache)

- `T_live`: TTL for a positive entry. This is a cache lifetime safety net only, NOT a
  correctness requirement: if it expires mid-activeflow, the next event simply misses and
  self-heals via fallback. Proposed default 24h (configurable). `activeflow_deleted`
  removes it earlier in the normal case.
- `T_neg`: TTL for a negative entry (activeflow has no webhook_uri, OR is deleted/not
  found). Short, e.g. 10m. Prevents a fallback RPC storm for the (common) case of
  activeflows that do NOT use a per-activeflow webhook.

### 5.4 Fallback result branching (revised round 2)

On cache miss, the fallback RPC `FlowV1ActiveflowGet(ctx, id)` result is classified:
- normal + webhook_uri set        → cache POSITIVE (T_live), deliver
- normal + webhook_uri empty       → cache NEGATIVE (T_neg), skip
- `tm_delete != nil` (soft-deleted)→ cache NEGATIVE (T_neg), skip. MUST NOT cache positive,
  to avoid resurrecting a dead activeflow's destination for up to T_live.
- NotFound (typed error)           → DO NOT cache, or cache NEGATIVE with a VERY short TTL
  (T_transient, e.g. 5s). Rationale (round 2): an event carrying this activeflow_id
  arrived, so the activeflow almost certainly exists; a NotFound here is most likely a
  flow-manager cache/replication lag, NOT a permanent absence. Caching it as a normal
  10m negative would silently drop per-activeflow delivery for up to 10m. Treat NotFound
  as transient and let the next event retry.
- RPC/transport error              → DO NOT cache, skip the extra delivery only. The
  customer URI delivery is independent and unaffected.

`FlowV1ActiveflowGet` on a soft-deleted activeflow returns the record with `tm_delete`
set (soft-delete pattern), so the `tm_delete != nil` branch is the primary guard; the
NotFound branch covers genuinely-unknown ids.

### 5.5 Single-flight on fallback (revised round 2)

Multiple events for the same new activeflow (e.g. call_created, call_updated,
recording_started) can arrive nearly simultaneously BEFORE `activeflow_created` is
consumed, each on its own goroutine. Without coordination they all issue a fallback RPC
for the same id (thundering herd). The fallback path MUST coalesce concurrent misses for
the same activeflow_id using `golang.org/x/sync/singleflight` (or an equivalent per-id
lock), so at most one in-flight RPC per id. NOTE: coalescing is per-pod only; with N
webhook-manager pods up to N concurrent fallback RPCs are possible. Acceptable at current
scale.

### 5.6 Cache write ordering / resurrection race (added round 2 — MAJOR)

`subscribehandler` processes each event on its own goroutine (`go h.processEvent`), and
RabbitMQ does not guarantee ordering across redelivery / multiple pods / prefetch. So
`activeflow_created` (SET positive) and `activeflow_deleted` for the same id may be
processed out of order. If `deleted` (DEL) runs first and `created` (SET positive) runs
after, the dead activeflow's destination is resurrected for up to T_live. This is the
SAME hazard 5.4 guards against, reached via a different path.

Mitigation (REQUIRED): make cache writes monotonic. Store the activeflow's `tm_update`
(or `tm_delete`) timestamp alongside the entry. A write only applies if its source event
timestamp is >= the stored one. Concretely:
- `activeflow_created` / `activeflow_updated` write a POSITIVE entry (T_live) when
  `webhook_uri` is set, otherwise a NEGATIVE entry (T_neg), carrying the event timestamp
  as the monotonic Tm (the lifecycle event carries `webhook_uri`, see 5.2 / Option A).
- `activeflow_deleted` writes a NEGATIVE tombstone (T_neg) carrying the delete timestamp,
  rather than a bare DEL.
- The fallback path (5.4) remains the lazy/miss safety net and also writes monotonically,
  SETting positive only if the resolved source timestamp is not older than the stored
  entry's timestamp.

The monotonic compare-and-set MUST be ATOMIC. A non-atomic GET-then-SET has a
read-modify-write race: two concurrent writers can both read the same stale timestamp and
both decide to write, so the older one can still clobber the newer one. The implementation
therefore performs the read+compare+set in a single Redis round trip via a Lua script
(`h.Cache.Eval`): the script GETs the stored entry, decodes its `tm` (stored as a
unix-nano integer so Lua can compare it numerically), and only `SET ... PX <ttl>` when the
incoming unix-nano `tm` is >= the stored `tm` (or the key is absent). This converts the
race into a self-correcting, atomic, last-writer-wins-by-timestamp scheme.

## 6. Data model changes

### 6.1 bin-flow-manager

`models/activeflow/activeflow.go` — add:
```go
WebhookURI    string        `json:"webhook_uri,omitempty"    db:"webhook_uri"`
WebhookMethod WebhookMethod `json:"webhook_method,omitempty" db:"webhook_method"`
```
Method is a TYPED enum, defined in flow-manager's own `models/activeflow` package (mirrors
customer-manager's `customer.WebhookMethod`):
```go
type WebhookMethod string
const (
    WebhookMethodNone   WebhookMethod = ""
    WebhookMethodPost   WebhookMethod = "POST"
    WebhookMethodGet    WebhookMethod = "GET"
    WebhookMethodPut    WebhookMethod = "PUT"
    WebhookMethodDelete WebhookMethod = "DELETE"
)
```
This follows the established pattern: each service keeps its OWN webhook-method type and
converts at the boundary. flow-manager defines its own `activeflow.WebhookMethod` and does
NOT import customer-manager's type. webhook-manager converts to its `webhook.MethodType`
when dispatching (same as it already does for the customer-level method via
`account.CreateAccountFromCustomer`). Validity is enforced both by the typed enum and at
the API layer (POST/PUT/GET/DELETE).

- `models/activeflow/field.go` — add `FieldWebhookURI`, `FieldWebhookMethod`.
- `models/activeflow/webhook.go` — `WebhookMessage` / `ConvertWebhookMessage` DO carry
  `webhook_uri` / `webhook_method` (Option A, see 5.2). These are the customer's OWN data
  returned to the customer's OWN endpoint and are documented response fields; the same
  `ConvertWebhookMessage()` powers BOTH the webhook event payload AND the REST response, so
  the fields are kept for consistency. webhook-manager pre-populates the cache from these
  fields and also uses the fallback RPC as a miss safety net.
  (`Matches()` uses reflect.DeepEqual, no change needed there.)
- `pkg/activeflowhandler/db.go Create(...)` — accept and persist the two new fields.
  DB persistence is automatic via `PrepareFields` reflection on `db` tags
  (`pkg/dbhandler/activeflow.go`), so only the model tag + Create wiring are needed.
- Migration: add columns to `flow_activeflows` (confirmed table name). New Alembic
  revision under `bin-dbscheme-manager/bin-manager/main/versions/`. Columns `varchar`,
  nullable/default empty. MUST be deployed BEFORE the code, else every activeflow
  INSERT fails on the missing column.

### 6.1.1 FlowV1ActiveflowCreate RPC signature change (CRITICAL SCOPE — added round 1)

`requesthandler.FlowV1ActiveflowCreate` is a shared `bin-common-handler` RPC. Adding the
two fields changes its signature and forces updates across the full call chain. To
contain blast radius we use a STRUCT/OPTIONS parameter rather than positional args.

Decision: extend the flow-manager activeflow-create REQUEST model (listenhandler request
struct + the `V1DataActiveFlowsPost`-style body) with optional `webhook_uri` /
`webhook_method`, and add them to the RPC as a backward-friendly options struct. The
following non-test callsites of `FlowV1ActiveflowCreate` must be reviewed and (if
signature changes) updated to pass empty values:

- `bin-common-handler/pkg/requesthandler/flow_activeflow.go` (definition), `main.go` (iface)
- `bin-call-manager/pkg/callhandler/outgoing_call.go`, `callhandler/start.go`,
  `recordinghandler/stop.go`
- `bin-ai-manager/pkg/summaryhandler/start.go`
- `bin-transcribe-manager/pkg/transcribehandler/stop.go`
- `bin-campaign-manager/pkg/campaignhandler/execute.go`
- `bin-conversation-manager/pkg/conversationhandler/message.go`
- `bin-api-manager/pkg/servicehandler/aicall.go`, `servicehandler/activeflows.go`

Only api-manager's activeflow-create path actually populates the new fields; all other
callers pass empty. Per root CLAUDE.md, a `bin-common-handler` signature change requires
running the full verification workflow in EVERY affected service (6 services here).

Mock regeneration required (`go generate ./...`):
- `bin-common-handler/pkg/requesthandler/mock_main.go` (FlowV1ActiveflowCreate)
- `bin-flow-manager/pkg/activeflowhandler/mock_main.go` (ActiveflowHandler.Create)
- `bin-api-manager/pkg/servicehandler` mock (ActiveflowCreate)
Plus updating every existing `EXPECT().FlowV1ActiveflowCreate(...)` and direct
`h.Create(...)` / `ActiveflowCreate(...)` test call (call-manager start_test.go,
outgoing_call_test.go, conversation/campaign/ai/api tests, flow-manager db_test.go).

`FlowV1ActiveflowGet` already EXISTS (`requesthandler/flow_activeflow.go`), used for the
fallback. No new RPC needed for the read path.

### 6.2 bin-webhook-manager

- `pkg/cachehandler` — add `ActiveflowWebhookGet/Set/Delete` (Redis key
  `webhook:activeflow:{id}`), storing `{uri, method}` or a negative marker.
- New dedicated `pkg/activeflowhandler` (separate from `accounthandler`) —
  `Get(ctx, activeflowID)` implementing cache → singleflight fallback → backfill with the
  5.4 branching. Depends on `cacheHandler` + `reqHandler`.
- `pkg/subscribehandler` WIRING (explicit, added round 1):
  - constructor `NewSubscribeHandler` gains the new activeflow handler dependency.
  - `processEvent` switch gains `publisher == flow-manager && (activeflow_created |
    activeflow_updated | activeflow_deleted)` cases; add a `publisherFlowManager` constant.
    `activeflow_created` / `activeflow_updated` pre-populate the cache from the event
    payload (the event carries `webhook_uri`, see 5.2 / Option A): POSITIVE when
    `webhook_uri` is set, NEGATIVE when empty, using the event timestamp as the monotonic
    Tm. `activeflow_deleted` writes the negative tombstone (id + tm_delete). The fallback
    path on the first resource event remains the lazy/miss safety net.
  - `cmd/webhook-manager/main.go`: `subscribeTargets` currently single
    `QueueNameCustomerEvent`; add `QueueNameFlowEvent` (=`bin-manager.flow-manager.event`).
    `runSubscribe` signature/wiring updated; pass cacheHandler + reqHandler to the new
    handler. (Note: existing `accounthandler` does NOT use cachehandler today; the new
    handler introduces the cache dependency.)
  - subscribehandler mocks regenerated.
- `cmd/webhook-manager/main.go run()` wiring (explicit, added round 2): today `cache` is
  passed only to `dbhandler.NewHandler(db, cache)`; no handler receives `cacheHandler`
  directly. The new `pkg/activeflowhandler` needs `cache` + `reqHandler`, so `run()` must
  construct it and thread it into both `runSubscribe` and the webhookhandler. `runSubscribe`
  / `runListen` signatures gain the new dependency.
- `pkg/webhookhandler` `SendWebhookToCustomer` — after the existing customer delivery,
  extract `activeflow_id` from the nested envelope (5.2), resolve via the new handler,
  deliver additionally. `webhookhandler` gains the new handler as a dependency; its mocks
  regenerated.

### 6.3 bin-api-manager + bin-openapi-manager (OpenAPI-first, explicit — round 1)

Order is mandatory: openapi-manager first, then api-manager.
- `bin-openapi-manager/openapi/openapi.yaml` response schema `FlowManagerActiveflow`
  (~line 4989): add `webhook_uri` / `webhook_method`.
- `bin-openapi-manager/openapi/paths/activeflows/main.yaml` POST requestBody: add the two
  optional fields (currently only id/flow_id/actions/variables).
- `cd bin-openapi-manager && go generate ./...`, then `cd bin-api-manager && go generate`.
- `bin-api-manager/server/activeflows.go` handler: parse `req.WebhookUri`/`WebhookMethod`,
  pass to `serviceHandler.ActiveflowCreate`. This changes `ActiveflowCreate` service
  signature (`pkg/servicehandler/main.go`, `activeflows.go`) and its mock.
- Validate `webhook_method` ∈ {POST,PUT,GET,DELETE} and `webhook_uri` is a syntactically
  valid http(s) URL at the API layer. Runtime SSRF protection already enforced at delivery
  by webhook-manager's `validateWebhookURL` + `newSafeHTTPClient`.
- Confirm update path does NOT accept webhook_uri/method (keeps uri immutable, so the
  cache can safely ignore `activeflow_updated`).

### 6.4 Docs (RST)

- `bin-api-manager/docsdev/source/` — activeflow struct (new fields), webhook overview
  (per-activeflow destination behavior, additive semantics).
- Per root CLAUDE.md, RST struct docs MUST match `WebhookMessage` (activeflow webhook.go),
  not the internal model.

## 7. Hot-path performance analysis

- Positive/negative cache means the steady-state hot path is **one Redis GET** per event
  that carries an activeflow_id. No flow-manager RPC in steady state.
- Fallback RPC only on genuine cache miss (event arrives before `activeflow_created` is
  consumed, or after a webhook-manager restart that lost... no — Redis is shared, so a
  pod restart does NOT lose the cache). Practically, miss ≈ event-ordering race only,
  bounded and self-healing via backfill.
- Negative cache is the key guard against RPC storms from the common
  "activeflow without webhook" case.

## 8. Failure modes

| Scenario | Behavior |
|----------|----------|
| activeflow has no webhook_uri | negative cache hit → skip, no extra delivery |
| event arrives before activeflow_created consumed | cache miss → fallback RPC → backfill → deliver |
| flow-manager unreachable on fallback | log error, skip extra delivery; customer URI delivery still succeeds (independent) |
| activeflow_deleted consumed | cache becomes a negative tombstone (5.6), not a bare delete; late events resolve negative or fall to miss→fallback |
| activeflow_id absent in payload | fall through; customer-only delivery (unchanged) |
| Redis unavailable | extra delivery best-effort fails; customer delivery path independent. Should not break existing behavior. |
| created/deleted processed out of order | monotonic timestamped writes (5.6): late created cannot overwrite newer delete tombstone |
| transient NotFound on fallback (flow-manager lag) | not cached as 10m negative; treated transient (5.4), next event retries |

## 8.1 Observability (added round 2)

Today `SendWebhookToCustomer` only logs failures; there is no delivery metric, and
customer-delivery vs activeflow-delivery failures would be indistinguishable. This change
MUST add (reusing the existing `webhook_manager` Prometheus namespace; avoid duplicate
metric names per bin-common-handler rule):
- `webhook_manager_delivery_total{destination="customer|activeflow", result="success|error"}`
- fallback resolution counter `..._activeflow_resolve_total{result="positive|negative|notfound|rpc_error"}`
- cache outcome counter `..._activeflow_cache_total{result="hit_positive|hit_negative|miss"}`
- structured logs on the activeflow-delivery path MUST include `activeflow_id` (the current
  `SendWebhookToCustomer` log fields do not).

## 9. Backward compatibility

Fully non-destructive. Activeflows without `webhook_uri` behave exactly as today
(customer-only delivery). No change to existing customer integrations. New fields are
optional throughout (API, model, payload via `omitempty`).

## 10. Test plan (unit only; no cost-incurring ops)

Mandatory fixture rule: ALL webhookhandler delivery tests MUST use the nested envelope
shape `{"type":"call_updated","data":{"activeflow_id":"..."}}` (matching the real wire
format, see 5.2 and existing `webhook_test.go:39`). A regression guard test MUST assert
that an `activeflow_id` placed at the TOP level (wrong shape) does NOT trigger the extra
delivery — this locks the round-1 blocker.

- flow-manager: Create persists new fields; `ConvertWebhookMessage` DOES carry them
  (Option A: customer's own data, documented response fields, see 5.2); update path
  rejects/ignores webhook_uri (immutability).
- webhook-manager cachehandler: positive/negative round-trip; tombstone with timestamp.
- webhook-manager activeflowhandler: cache hit (positive/negative), miss→fallback→deliver,
  miss→fallback(empty uri)→negative, miss→fallback(tm_delete!=nil)→negative (no positive),
  miss→fallback(NotFound)→transient (not 10m negative), miss→fallback(rpc error)→no cache.
- single-flight: concurrent misses for one id issue exactly one fallback RPC.
- monotonic writes (5.6): out-of-order created-after-deleted does NOT resurrect positive.
- webhook-manager subscribehandler: created/updated pre-populate the cache from the event
  payload (positive when webhook_uri present, negative when empty), deleted writes the
  tombstone (id + tm_delete), out-of-order resilience.
- webhook-manager webhookhandler: SendWebhookToCustomer with
  (a) no activeflow_id (customer-only), (b) positive hit (additive), (c) negative hit
  (customer-only), (d) miss→fallback→additive, (e) miss→fallback error→customer-only,
  (f) Redis down → customer delivery still succeeds,
  (g) TOP-LEVEL activeflow_id regression guard (no extra delivery).
- self-delivery: an activeflow lifecycle event (id, not activeflow_id) does NOT trigger
  per-activeflow delivery.
- metrics: delivery/resolve/cache counters increment on the correct label paths.
- api-manager: request validation for method/uri.

## 11. Resolved questions

1. Handler placement: NEW dedicated `pkg/activeflowhandler` in webhook-manager (not
   extending accounthandler). RESOLVED.
2. `requesthandler.FlowV1ActiveflowGet` EXISTS — used for fallback, no new RPC. RESOLVED.
3. `activeflow_updated` and cache: uri is immutable (API rejects changes), so updated does
   not change the cached uri. It MAY refresh the positive entry's TTL (sliding) to reduce
   fallback load on long-lived activeflows (optional optimization). RESOLVED.
4. T_live (default 24h), T_neg (default 10m), T_transient (default 5s) — all config flags.
   RESOLVED (values may be tuned in implementation).
5. Method storage: plain string on activeflow; validated at API layer. RESOLVED.
```

