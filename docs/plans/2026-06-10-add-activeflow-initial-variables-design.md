# Activeflow Initial Variables Injection

Design document. 2026-06-10. (v3 after review round 2)

## Problem Statement

When a new call (or any activeflow) is originated programmatically, there is currently
no way to seed runtime context into the resulting activeflow. The `create_call` LLM tool
(PR #976) lets the AI place an independent outbound call, but the originated flow starts
with an empty variable set plus only the system-reserved keys. The natural next requirement
("AI가 콜을 걸 때 컨텍스트를 새 콜의 flow에 실어 보낸다", e.g. campaign id, caller intent,
correlation key) cannot be met today.

### Why `Call.Metadata` is the wrong channel

`CallV1CallsCreate`'s last argument `metadata map[string]any` maps to `Call.Metadata`, which
is a **closed registry** (`bin-call-manager/models/call/metadata.go` `ValidMetadataKeys`):
only `rtp_debug`, `route_provider_ids`, `skip_source_validation`, `codecs`. call-manager's
listenhandler rejects unknown keys. It is internal call-control plumbing, NOT a business-data
bag. It is also call-scoped, not activeflow-scoped.

### The right target: activeflow Variables

An activeflow consumes runtime key-values (`map[string]string`) that a flow reads via
variable substitution. Activeflow has no "metadata" field; it has Variables.

**CRITICAL substitution syntax (verified in code):** the substitution token is
**`${var_name}`**, NOT `{{var_name}}`. The engine regex is
`bin-flow-manager/pkg/variablehandler/main.go`:
`regexVariable = regexp.MustCompile(\`\${(.*?)}\`)`. All user-facing copy, REST examples, and
the create_call tool description MUST use `${...}`. A customer who writes `{{campaign_id}}`
would get NO substitution (the literal text survives). This was a v1 documentation error
caught in review.

### The blocked chain

```
create_call / POST /calls / POST /activeflows
  → CallV1CallsCreate(...)               [no variables arg]
    → call-manager CreateCallsOutgoing/CreateCallOutgoing
      → FlowV1ActiveflowCreate(...)       [no variables arg]
        → activeflowHandler.Create(...)   [no variables arg]
          → variableCreate()              [seeds from inheritance + system-reserved only]
```

`variableCreate` (`bin-flow-manager/pkg/activeflowhandler/variable.go`) seeds variables from
exactly two sources today: (a) inheritance (if `ReferenceActiveflowID != Nil`, `maps.Copy`
the parent activeflow's variables); (b) system-reserved keys applied LAST (overwrite). The
reserved keys are: `voipbin.activeflow.id`, and the bare keys for reference_type,
reference_id, reference_activeflow_id, flow_id, complete_count (note: only some are under the
`voipbin.` prefix; see Reserved-Key Protection below). No external caller can say "start with
key=value".

## Scope

### In scope (Phase 1)

- Thread `variables map[string]string` through the full origination chain:
  `CallV1CallsCreate` and `FlowV1ActiveflowCreate` (both `bin-common-handler/pkg/requesthandler`).
- Merge externally-supplied variables in `variableCreate` BETWEEN inheritance and the
  system-reserved overwrite (see Merge Order).
- Expose `variables` on three user-facing entry points:
  1. REST `POST /calls`
  2. REST `POST /activeflows`
  3. `create_call` LLM tool (bin-ai-manager)
- Sanitize/validate externally-supplied variables in `variableCreate` (the sanitizing
  authority), with a fail-closed early-reject (HTTP 400) at the api-manager edge for
  user-facing paths.

### Out of scope (signature propagation only, pass nil)

Adding the trailing parameter to the two shared-lib signatures is a source-breaking change:
EVERY call site must be updated to compile. The following production call sites have no
user-supplied variables today and pass `nil`. (This list was 14 sites at design time;
**verify by grep at implementation start** — the shared lib has 37 consumers and the count
drifts.)

`CallV1CallsCreate` production sites — 6 total (verified by grep, non-test):
- bin-api-manager/pkg/servicehandler/call.go (POST /calls — **DOES forward variables**, the user path)
- bin-queue-manager/pkg/queuecallhandler/execute.go (pass nil)
- bin-route-manager/pkg/providercallhandler/providercall.go (pass nil)
- bin-flow-manager/pkg/activeflowhandler/actionhandle.go (connect action internal originate — pass nil)
- bin-flow-manager/pkg/activeflowhandler/actionhandle.go (second connect site — pass nil)
- bin-ai-manager/pkg/aicallhandler/tool.go (connect_call path nil; create_call path **DOES forward variables**)

(Correction from v2: there is NO "bin-call-manager internal originate" CallV1CallsCreate
caller — call-manager is the RECEIVER of this RPC, not a caller. That ghost entry was removed.)

`FlowV1ActiveflowCreate` production sites — 9 total (verified by grep, non-test):
- bin-api-manager/pkg/servicehandler/activeflows.go (POST /activeflows — **DOES forward variables**, user path)
- bin-api-manager/pkg/servicehandler/aicall.go (AIcallCreate / POST /aicalls — **pass nil**; does NOT expose variables but MUST get the nil arg or it fails to compile. This site was MISSING from v2 and is the one source-breaking gap an implementer would hit.)
- bin-call-manager/pkg/callhandler/outgoing_call.go (outbound — **DOES forward variables**)
- bin-call-manager/pkg/callhandler/start.go (inbound call activeflow — pass nil)
- bin-call-manager/pkg/recordinghandler/stop.go (on_end_flow — pass nil)
- bin-campaign-manager/pkg/campaignhandler/execute.go (pass nil)
- bin-conversation-manager/pkg/conversationhandler/message.go (pass nil)
- bin-transcribe-manager/pkg/transcribehandler/stop.go (pass nil)
- bin-ai-manager/pkg/summaryhandler/start.go (pass nil)

Plus all `*_test.go` mock expectations and regenerated `mock_main.go` in every consumer.
**At implementation start, re-grep both RPC names with `!*_test.go` to confirm the count has
not drifted** (37-consumer shared lib).

### Out of scope (deferred)

- Per-resource variable injection UX for campaign/conversation/transcribe.
- Mutating variables on an already-running activeflow via this path (use existing
  `set_variables` action / `FlowV1VariableSetVariable` RPC).

## API surface choice: positional param vs options struct (Open Question E)

Review round 1 (reviewer 2) flagged the source-breaking signature change and proposed an
options struct / new method (`CallV1CallsCreateWithVariables`) to shrink blast radius.

**Recommendation: keep the positional trailing param, do NOT introduce an options struct.**
Rationale:
- VoIPBin's established convention is unwrapped positional parameters on requesthandler RPC
  clients (the same convention that drove the create_call PR #976 to pass `metadata` as a
  positional `nil`). An options struct here would be an inconsistent one-off.
- The blast radius is mechanical (add `nil` at ~13 sites + regenerate mocks), caught at
  compile time, and fully covered by the monorepo verification workflow. It is not a
  correctness risk, only labor.
- A parallel `*WithVariables` method permanently forks the call path and doubles the
  maintenance surface for every future change to call origination.

This is surfaced as Open Question E for CTO ratification, since it is a deliberate
accept-the-blast-radius decision.

## Domain Model

No new entity. The transported value is `map[string]string` end to end, matching the existing
`variable.Variable.Variables` field type (`bin-flow-manager/models/variable/variable.go`).

```go
// variable.Variable (existing, unchanged)
type Variable struct {
    ID        uuid.UUID         `json:"id"`        // == activeflow id
    Variables map[string]string `json:"variables"`
}
```

### Storage note

Variables are stored in **Redis only** (`dbhandler.VariableCreate` → `variableSetToCache`;
no MySQL table). This is why the size limits below are operational (read-amplification and
Redis memory) rather than a column-type ceiling.

## Validation & Sanitization Rules (the contract)

Two enforcement points:
1. **api-manager edge (fail-closed, HTTP 400)** for user-facing paths (`POST /calls`,
   `POST /activeflows`). Returns 400 BEFORE the RPC, so a user gets a synchronous reject.
2. **flow-manager `variableCreate` (sanitizing authority)** for ALL paths, including internal
   RPC callers that bypass the edge.

| Rule | Limit | Applied |
|---|---|---|
| Reserved-prefix key (normalized) | dropped | per-key, BEFORE counting |
| Empty key (after trim) | the key is dropped | per-key |
| Max per-value bytes (UTF-8) | 32 KB | per-value; oversize value → reject injection |
| Max external key count (after drops) | 100 | whole injection |
| Max external total bytes (after drops, keys+values UTF-8) | 64 KB | whole injection |
| Max MERGED total bytes (inheritance + external + reserved) | 256 KB | secondary cap, see below |

### Order of operations inside the sanitizer (precise)

1. For each externally-supplied key: trim surrounding whitespace; normalize for the reserved
   check (ASCII-lowercase + trim, comparison-only — the stored key keeps its original case).
   **Drop the key (silently) if EITHER** (a) the normalized form starts with `voipbin.`, **OR**
   (b) the normalized form exactly matches one of the bare reserved keys (`reference_type`,
   `reference_id`, `reference_activeflow_id`, `flow_id`, `complete_count`). If the trimmed key
   is empty → drop. This closes the bare-reserved-key gap (esp. `complete_count`, see Merge
   Order). Normalization is for COMPARISON only; it never mutates the stored key, so a
   legitimate key like `CustomerRegion` is stored verbatim and `${CustomerRegion}` still
   substitutes.
2. Per-value byte check on the survivors; if any value > 32 KB → reject the whole external
   injection (do NOT silently truncate).
3. Count survivors: if key count > 100 OR external total bytes > 64 KB → reject the whole
   external injection.
4. The counting in step 3 happens AFTER the reserved/empty drops, so a caller cannot pad with
   `voipbin.*` junk keys to push legitimate keys over the limit.

### Merged-size secondary cap (defense-in-depth, with honest reachability)

The 64 KB external cap bounds a single injection. A theoretical concern was cumulative growth
across an `on_complete_flow` chain (depth bound `maxActiveflowCompleteCount = 5`). **Code
verification (round 2) showed this worst case is effectively unreachable today:** the
`on_complete_flow`-derived and inbound/internal activeflow-creation paths all pass `nil`
variables (only the three user-facing origination entry points inject), so a chain does not
add a fresh 64 KB per generation. Inheritance copies the FIRST generation's external set
forward without growing it. So the realistic merged size is bounded by one injection (~64 KB)
plus reserved keys, well under the 256 KB cap (which is 4 × 64 KB).

The 256 KB merged cap is therefore **defense-in-depth, not load-bearing** — it guards against a
FUTURE change that starts injecting variables on a derived/internal path (which would
reintroduce the cumulative vector). The rule is:

- `variableCreate` computes the FINAL merged map byte size. If it exceeds 256 KB, the
  EXTERNAL injection for this generation is dropped (inheritance + reserved preserved) and an
  Error is logged with `outcome=rejected_merged`.
- **If inheritance ALONE already exceeds 256 KB** (only possible via the future-vector above),
  dropping external cannot bring it under the cap. In that case the activeflow is STILL created
  (inheritance + reserved preserved, never reserved-less) and an Error is logged — the cap
  cannot retroactively shrink an already-oversized inherited set, and silently truncating
  inherited business variables would be worse (corrupts an in-flight flow). This is logged-but-
  permitted, and is unreachable on the Phase-1 paths. Documented so an implementer adding a
  derived-path injection later sees the boundary explicitly.

Net: on Phase-1 paths the cap never fires; it exists so the invariant is stated for future
maintainers, not because the current paths can hit it.

### Byte counting

UTF-8 byte length (`len(s)` on the Go string), not rune count.

## Merge Order (load-bearing)

`variableCreate` builds the final map in exactly this order:

```
1. parent inheritance   (maps.Copy of parent activeflow variables, if ReferenceActiveflowID != Nil)
2. external injection    (the sanitized initialVariables: reserved-dropped, validated)
3. system-reserved       (voipbin.activeflow.id, reference_type, reference_id,
                          reference_activeflow_id, flow_id, complete_count) — applied LAST
```

Rationale:
- Explicit external input (2) overrides implicit inheritance (1): a caller stating
  `key=value` should win over a copied parent value. Intuitive precedence.
- System-reserved (3) is always last so a customer cannot forge system identity/control
  variables. This LAST-overwrite is the PRIMARY forgery defense (see Reserved-Key Protection).

**External-overrides-inheritance trust note (round-2 G3):** because step 2 beats step 1, a
derived activeflow's externally-supplied variable can overwrite a NON-reserved business
variable that the parent set (e.g. a parent's `${authorized_amount}`). This is bounded and
acceptable for Phase 1 for three reasons: (a) it is strictly same-customer — `customerID`
flows from the originating record, never from caller input, so there is no cross-tenant
override; (b) on the Phase-1 paths the override only happens when a caller EXPLICITLY supplies
that key, which is the documented precedence, not a silent surprise; (c) critically, the
`create_call` (LLM) path has `ReferenceActiveflowID == Nil`, so it has NO inheritance to
override at all — the lower-trust LLM source can never clobber a parent variable. The override
concern only applies to the REST paths (`POST /calls` / `POST /activeflows` with an explicit
`reference_activeflow_id`), where the caller is the customer's own authenticated integration,
not the LLM. If a future change lets a lower-trust source inherit AND inject on the same path,
revisit this (candidate: namespace external vars, or make inherited non-reserved keys
override-protected). Tracked as Open Question H.

For `create_call`'s independent origination, `ReferenceActiveflowID == Nil`, so step 1 is a
no-op; only steps 2 and 3 apply. For `on_complete_flow`-derived activeflows, all three apply.

**`complete_count` parse timing (must not regress) — corrected rationale:** the existing code
parses `complete_count` from the INHERITED map (step 1 result) BEFORE the reserved overwrite,
to increment it and enforce `maxActiveflowCompleteCount = 5`. `complete_count` is a BARE
reserved key (NOT under the `voipbin.` prefix), so the `voipbin.` sanitizer drop does NOT
remove an externally-supplied `complete_count`. The v2 claim "external cannot affect because
voipbin.* dropped" was WRONG for this key.

The actual safety property is the READ SOURCE: the depth counter must be parsed from the
inherited map only, never from the post-step-2 merged map. If an implementer reads
`complete_count` from the merged map, an attacker-supplied `complete_count=0` in the external
injection would reset the depth counter every generation, DEFEATING the depth-5 bound and
enabling unbounded `on_complete_flow` inheritance growth (which also defeats the merged-size
cap). Two hard requirements:
1. Parse `complete_count` from the step-1 (inherited) map, before step 2 is applied.
2. Treat any `complete_count` (or any other bare reserved key) present in the EXTERNAL
   injection as a reserved-key collision: drop it in sanitization (extend the drop set beyond
   the `voipbin.` prefix to the explicit bare reserved-key list), so external input can never
   carry a reserved key at all. This makes the read-source requirement defense-in-depth rather
   than the sole guard.

A regression test must pin both: (a) the increment reads the inherited value with step-2
inserted between inheritance and the reserved overwrite, and (b) an external `complete_count`
is dropped and cannot influence the depth bound.

## Reserved-Key Protection (three legs — do not conflate)

The reserved keys are NOT all under the `voipbin.` prefix. Verified set:
`voipbin.activeflow.id` (prefixed) plus the bare reference_type/reference_id/
reference_activeflow_id/flow_id/complete_count keys.

- **PRIMARY defense (covers ALL reserved keys):** step 3 applies the system-reserved keys
  LAST, overwriting any external value with the same key. This is what actually prevents
  forgery of every reserved key, prefixed or not, and it is the load-bearing guarantee — a
  regression test MUST pin "reserved keys are applied after external injection".
- **SECONDARY hygiene (sanitizer drop):** step 1 of sanitization drops any external key that
  (a) normalizes (case-insensitive + trim) to a `voipbin.`-prefix, OR (b) exactly matches a
  bare reserved key (`reference_type`/`reference_id`/`reference_activeflow_id`/`flow_id`/
  `complete_count`). This prevents reserved keys from ever entering the external map, so logs
  and future consumers never see a near-reserved look-alike, and (critically for
  `complete_count`) the depth counter cannot be seeded from external input. Drop is silent
  (not an error) since a caller may innocently echo a variable dump.

The three together: the LAST-overwrite is the forgery guarantee (leg 1); the sanitizer drop is
defense-in-depth that also closes the bare-`complete_count` look-alike vector (leg 2); the
read-source rule (parse `complete_count` from the inherited map only, see Merge Order) is the
depth-bound guarantee (leg 3). The depth bound does NOT rely on any single leg.

## Failure Semantics (resolves the v1 fail-closed-vs-non-fatal contradiction)

The phrase "fail-closed authority" in v1 was contradictory with the existing non-fatal
`variableCreate`. Corrected model:

- **User-facing paths**: api-manager edge validates and returns **HTTP 400** before the RPC.
  This is the only true fail-closed reject, and it is synchronous and user-visible.
- **`variableCreate` (internal authority)**: NEVER fails activeflow creation over external
  variables, and NEVER produces a reserved-less activeflow. On any external-variable problem
  (oversize value, count/byte cap, merged cap), it **drops the external injection for that
  generation, still seeds inheritance + reserved keys, and logs Error**. The activeflow is
  always created with a valid reserved-key set.
- **Redis write failure**: unchanged — non-fatal, logged (existing contract). Note the
  invariant "NEVER produces a reserved-less activeflow" is about the SANITIZER's behavior over
  external variables (it always still seeds inheritance + reserved into the map it builds). It
  is NOT a guarantee that the Redis write of that map succeeds: if `variableSetToCache` fails,
  the activeflow exists with no variable record at all, exactly as today. That pre-existing
  failure mode is out of scope for this feature and unchanged; the invariant is scoped to "the
  map the sanitizer hands to the cache always contains the reserved set", not "the cache write
  cannot fail".

Verified nuance: `Create` (`db.go`) only dereferences the returned variable in the `else`
(success) branch, so the current error path does not nil-deref. The implementation must
PRESERVE that guard when `variableCreate` gains the drop-and-log path (do not move the
`v.ID` access out of the success branch).

### Consequence for `create_call` (LLM) — Open Question F

Because internal paths drop-and-log (not 400), an LLM emitting 100 garbage keys or an oversize
value results in a SILENTLY dropped injection: the call still places, but the seed context is
lost with only a server-side Error log. Two mitigations, decide in review:
- (a) Add the Phase-2 counter (below) now, scoped minimal, so dropped injections are at least
  observable from the AI path.
- (b) Keep create_call's "no validation beyond JSON shape" stance (flow-manager is authority),
  accept silent drop as the failure mode, rely on the log + counter.

Recommendation: (b) + ship the counter in Phase 1 (cheap, closes the observability blind spot).

## Handler Interface Changes

### bin-common-handler/pkg/requesthandler (shared, 37-service blast radius)

```go
// CallV1CallsCreate — add trailing `variables map[string]string`
func (r *requestHandler) CallV1CallsCreate(
    ctx context.Context, customerID, flowID, masterCallID uuid.UUID,
    source *commonaddress.Address, destinations []commonaddress.Address,
    earlyExecution, connect bool, anonymous string,
    metadata map[string]any,
    variables map[string]string,           // NEW
) ([]*cmcall.Call, []*cmgroupcall.Groupcall, error)

// FlowV1ActiveflowCreate — add trailing `variables map[string]string`
func (r *requestHandler) FlowV1ActiveflowCreate(
    ctx context.Context, activeflowID, customerID, flowID uuid.UUID,
    referenceType fmactiveflow.ReferenceType, referenceID, referenceActiveflowID uuid.UUID,
    variables map[string]string,           // NEW
) (*fmactiveflow.Activeflow, error)
```

Both marshal the new field into their respective listenhandler request DTOs.

### Wire DTOs (omitempty so nil marshals away, old↔new compatible)

```go
// bin-call-manager .../request/calls.go  V1DataCallsPost — add:
Variables map[string]string `json:"variables,omitempty"`

// bin-flow-manager .../request/activeflows.go  V1DataActiveFlowsPost — add:
Variables map[string]string `json:"variables,omitempty"`
```

Wire compatibility: a nil Go map with `omitempty` is dropped from JSON entirely; the receiver
unmarshals absent/null → nil, `{}` → empty non-nil. The sanitizer treats nil and empty
identically (range over nil is a no-op). A regression test must cover all three shapes
(absent / null / `{}`) to lock rolling-deploy compatibility between old and new flow-manager.

### bin-flow-manager activeflowHandler

```go
func (h *activeflowHandler) Create(
    ctx context.Context, id, customerID uuid.UUID,
    referenceType activeflow.ReferenceType, referenceID, referenceActiveflowID, flowID uuid.UUID,
    initialVariables map[string]string,    // NEW
) (*activeflow.Activeflow, error)

func (h *activeflowHandler) variableCreate(
    ctx context.Context, af *activeflow.Activeflow, initialVariables map[string]string,
) (*variable.Variable, error)
```

### bin-call-manager propagation

`CreateCallsOutgoing` / `CreateCallOutgoing` (`pkg/callhandler/outgoing_call.go`) accept the
variables and forward them to the `FlowV1ActiveflowCreate(..., ReferenceTypeCall, id,
uuid.Nil, variables)` call at line ~234. The inbound-call site in `start.go` and the
on_end_flow site in `recordinghandler/stop.go` pass nil.

## REST API

### POST /calls (add optional `variables`)

```json
{
  "flow_id": "1b2c...",
  "destinations": [{"type": "tel", "target": "+155****4567"}],
  "variables": { "campaign_id": "summer-2026", "customer_name": "Jane Doe", "intent": "renewal_reminder" }
}
```
A flow action reads `${campaign_id}`, `${customer_name}`, etc.

### POST /activeflows (add optional `variables`)

```
{ "flow_id": "1b2c...", "reference_type": "none", "variables": { "ticket_id": "T-9921" } }
```

**Reserved keys (silently ignored if supplied in `variables`):** any key starting with
`voipbin.`, plus the exact keys `flow_id`, `reference_type`, `reference_id`,
`reference_activeflow_id`, `complete_count`. These are system-managed; supplying them has no
effect (they do not error, they are dropped before substitution). Hard-limit violations
(count/size) DO return 400; reserved-key collisions do not (a caller may innocently echo a full
variable dump).

### Error response (edge validation, 400)

```
{ "message": "too many variables (max 100)" }
{ "message": "variables total size exceeds 64KB" }
{ "message": "variable value 'notes' exceeds 32KB" }
```

## create_call Tool Schema (bin-ai-manager)

Add a `variables` property to the `create_call` tool definition
(`pkg/toolhandler/definitions.go`):

```
"variables": {
  "type": "object",
  "description": "Optional flat key-value context seeded into the new call's flow as runtime
    variables, readable in the flow via ${key}. String values only. Reserved keys are ignored:
    any key starting with 'voipbin.' and the exact keys flow_id, reference_type, reference_id,
    reference_activeflow_id, complete_count. Max 100 keys, 64KB total.",
  "additionalProperties": { "type": "string" }
}
```

`toolHandleCreateCall` (`pkg/aicallhandler/tool.go`) parses the variables and passes them as
the new trailing arg of `CallV1CallsCreate`.

**LLM type coercion (review finding):** an LLM frequently emits non-string values
(`{"campaign_id": 123}`). Unmarshaling directly into `map[string]string` would fail and abort
the whole tool call. Parse into `map[string]any` and coerce scalar values (string/number/bool)
to their string form; skip/replace non-scalar (object/array) values with a logged note. This
keeps a slightly-malformed LLM emission from killing an otherwise valid call.

LLM-facing copy uses `${...}` and the terms activeflow / flow, never "call flow".

## Security & Compliance

- **Reserved namespace integrity**: three-leg defense above (LAST overwrite + sanitizer drop of
  `voipbin.`-prefix and bare reserved keys + inherited read-source for the depth counter). A
  customer cannot forge reserved keys.
- **Variable substitution is an injection surface (CTO attention).** Variables are substituted
  via `${}` into downstream action options, including `talk` text and, notably, `webhook_call`
  URLs/bodies. Since `create_call` lets an LLM author variable VALUES, AI-generated strings can
  flow into outbound HTTP request URLs/bodies → a template/SSRF-adjacent surface that did not
  previously accept AI-authored content on this path. This is NOT introduced by this feature
  alone (any `set_variables` + `webhook_call` flow already has it), but this feature widens the
  set of who can write those values to "the LLM during a call". Phase-1 position: document the
  risk; the existing webhook_call egress controls (customer-configured destinations) remain the
  boundary. Flagged as Open Question G for CTO: do we want an allowlist/escaping pass on
  variable values that reach URL-typed action options. Recommendation: separate hardening
  effort, not a blocker for this feature.
- **No IDOR surface added**: variables are opaque strings, not resource references. The
  `flow_id` ownership check in `create_call` and api-manager customer scoping are unchanged.
- **DoS bound**: per-injection (64 KB / 100 keys / 32 KB-value) + merged (256 KB) caps, plus
  chain depth bound (`maxActiveflowCompleteCount = 5`). Fail-closed at the edge; sanitize-and-
  drop at the internal authority. A direct RPC caller cannot exceed the stored-size cap.
- **PII**: variable values may contain customer PII (names, intents). They live in Redis with
  the activeflow lifetime (same as today's gather-captured variables) — no new external egress,
  no new LLM exposure beyond what the customer's own flow already does. No GDPR posture change.

## Observability

Ship one counter in Phase 1 (closes the silent-drop blind spot on the AI path):
`flow_activeflow_variable_injection_total{outcome}` where outcome ∈
`accepted | dropped_reserved | rejected_value_size | rejected_count | rejected_total |
rejected_merged`. The `outcome` label is the ONLY label and its value set is a FIXED enum —
NEVER use the variable key, value, customer_id, or any caller-controlled string as a label
(Prometheus cardinality safety). `variableCreate` logs Debug on reserved drop and Error on any
reject-and-drop (with activeflow_id + failing rule). No latency histogram needed (synchronous
map work).

## Affected Services

| Service | Change | Exposes input? |
|---|---|---|
| bin-common-handler | `CallV1CallsCreate` + `FlowV1ActiveflowCreate` signatures + regenerated mocks | n/a (transport) |
| bin-flow-manager | `V1DataActiveFlowsPost.Variables`, `Create`, `variableCreate` sanitize+merge, counter | n/a (origin) |
| bin-call-manager | `V1DataCallsPost.Variables`, outgoing propagation, inbound/on_end_flow pass nil | n/a (transit) |
| bin-ai-manager | `create_call` schema + `toolHandleCreateCall` (coerce); summaryhandler/connect_call pass nil | yes (create_call) |
| bin-api-manager | `POST /calls`, `POST /activeflows` body `variables` + edge 400 validation; `AIcallCreate` FlowV1ActiveflowCreate call site → pass nil | yes (calls/activeflows only) |
| bin-openapi-manager | OpenAPI: `variables` on both request bodies | yes (doc) |
| bin-queue-manager, bin-route-manager, bin-campaign-manager, bin-conversation-manager, bin-transcribe-manager | shared-sig call sites → pass nil | no |

## Implementation Order

1. **bin-flow-manager origin first**: DTO field; `variableCreate(ctx, af, initialVariables)`
   with sanitize (reserved-drop normalized, empty-drop, per-value, count, total, merged cap) +
   3-step merge + counter; `Create(... initialVariables)`; listenhandler reads `req.Variables`;
   PRESERVE the `db.go` success-branch `v.ID` guard. Tests: sanitize matrix, merge precedence,
   complete_count timing, reserved normalization (case/space), merged cap, nil/null/`{}` wire.
2. **bin-common-handler**: both signatures marshal `Variables`; regenerate mocks; widen all
   call-site gomock matchers to new arity.
3. **bin-call-manager**: DTO; outgoing propagation; inbound + on_end_flow nil.
4. **bin-ai-manager**: create_call schema + parse/coerce + pass; connect_call/summary nil.
5. **bin-api-manager**: REST `variables` + edge 400 validation; pass through.
6. **bin-openapi-manager**: spec; regenerate; sync to JS (voipbin-openapi-sync-to-js).
7. **nil-propagation services**: compile fixes.
8. Full verification workflow in EVERY touched service.

## Open Questions

| # | Question | Recommendation | Owner / Priority |
|---|---|---|---|
| A | (resolved) fail-closed vs non-fatal | Edge 400 for users; sanitize-and-drop internally, reserved always seeded, never fail create | Resolved in v2 |
| B | Reserved prefix: only `voipbin.`? | Yes (matches the single existing reserved namespace); normalize case+space | CPO / Decided unless objection |
| E | Positional param vs options struct? | Positional (VoIPBin convention; mechanical blast radius) | CTO / Pre-implementation |
| F | create_call silent-drop on bad LLM vars | Keep no-edge-validation, ship counter in Phase 1 for observability | CTO / Pre-implementation |
| G | Escaping/allowlist for vars reaching URL-typed action options (SSRF surface) | Separate hardening effort, not a blocker | CTO / Phase 2 |
| H | External-overrides-inheritance: should a lower-trust source be allowed to override a parent's non-reserved business variable? | Phase-1 paths are safe (LLM has no inheritance; REST callers are the customer's own integration). Revisit only if a future path inherits AND injects from a lower-trust source | CTO / Future |
| D | Expose `variables` for campaign/conversation/transcribe origination? | Separate per-resource effort | CPO / Future |

## Review Summary (round 1 → v2)

Two independent reviewers (general + adversarial), both CHANGES REQUESTED, both read the
actual codebase. Changes applied in v2:

- **Critical: substitution syntax** corrected `{{var}}` → `${var}` everywhere (verified
  `regexVariable = \`\${(.*?)}\``). v1 would have shipped a non-substituting syntax in docs.
- **Critical: failure-mode contradiction** resolved. "fail-closed authority" was incompatible
  with the non-fatal `variableCreate`. New model: edge 400 for users; internal sanitize-and-
  drop that NEVER produces a reserved-less activeflow and never fails creation. Preserve the
  `db.go` success-branch `v.ID` guard.
- **High: blast radius** corrected from "5 services" to the real ~13 call sites enumerated,
  with a "verify by grep at implementation start" caveat.
- **High: merged-size gap** closed with a 256 KB post-merge secondary cap (per-injection 64 KB
  did not bound cumulative inheritance across the depth-5 chain).
- **High: reserved-prefix bypass** (case/whitespace/lookalike) closed by normalizing
  (lowercase+trim) before the `voipbin.` check; clarified that LAST-overwrite is the real
  forgery defense and prefix-drop is namespace hygiene.
- **High: SSRF/template surface** of AI-authored variable values reaching webhook URLs
  documented (Open Question G).
- **Medium: LLM type coercion** for non-string values in create_call added.
- **Medium: validation order** specified (drop reserved/empty BEFORE counting).
- **Observability**: promoted the counter into Phase 1 to close the AI-path silent-drop blind
  spot.
- Open Questions A resolved; E (API shape), F (create_call drop), G (SSRF) added.

## Review Summary (round 2 → v3)

Two independent round-2 reviewers (full document inlined; one pure-reasoning, one with live
codebase access), both CHANGES REQUESTED. Code-grounded findings fixed in v3:

- **Critical: `complete_count` rationale was wrong.** It is a BARE reserved key, so the
  `voipbin.` prefix drop did NOT protect it; an external `complete_count=0` could reset the
  depth counter and defeat the `maxActiveflowCompleteCount=5` bound. Fixed: (1) the real guard
  is the READ SOURCE (parse from inherited map, before step 2); (2) the sanitizer drop set now
  explicitly includes the bare reserved keys, not just the `voipbin.` prefix. Regression tests
  pin both.
- **Compile-blocking: missing call site.** `bin-api-manager/servicehandler/aicall.go`
  (`AIcallCreate`) calls `FlowV1ActiveflowCreate` and was absent from v2's propagation list —
  it would fail to compile. Added to the nil-propagation list and Affected Services table. Also
  removed a v2 GHOST entry ("bin-call-manager internal originate" CallV1CallsCreate caller —
  call-manager is the receiver, not a caller). Full verified caller lists (6 + 9) now enumerated.
- **256 KB merged cap honesty.** Code review confirmed derived/internal paths pass nil
  variables, so the 5×64 KB cumulative worst case is unreachable on Phase-1 paths. The cap is
  re-scoped as defense-in-depth for a future derived-path injection, with explicit behavior
  documented for the (unreachable today) inheritance-alone-exceeds-cap case (logged, permitted,
  never reserved-less). No longer overstated as load-bearing.
- **G3 external-overrides-inheritance trust.** Documented why Phase-1 is safe (LLM path has no
  inheritance; REST callers are the customer's own authenticated integration; strictly
  same-customer). Added Open Question H for the future lower-trust-inherit-and-inject case.
- **Reserved-key normalization clarified** as comparison-only (stored key keeps original case),
  so legitimate mixed-case keys still substitute.
- **N5: reserved-less invariant vs Redis write failure** scoped precisely (invariant is about
  the sanitizer's built map, not the cache write succeeding).
- **Counter cardinality** locked to a single fixed-enum `outcome` label; no caller-controlled
  label values.
- Resolved round-1 items (syntax `${}`, fail-mode, blast-radius positional decision,
  reserved-prefix bypass, SSRF doc, LLM coercion, validation order) re-confirmed intact.

## Review Summary (round 3 → final)

Two round-3 reviewers: one APPROVE (go/no-go: ready to implement, no blockers outside Open
Questions), one CHANGES REQUESTED on documentation-consistency only (design substance sound).
The latter's findings, all fixed in this final revision:

- **H1: "two mechanisms" vs "three legs" mismatch.** The complete_count fix added a third
  defense leg (inherited read-source) but the header/Security still said "two/dual". Unified to
  "three legs" (LAST-overwrite forgery guarantee + sanitizer drop + inherited read-source) in
  the Reserved-Key Protection header and the Security section.
- **H2: user-facing docs omitted the bare-reserved-key drop.** The create_call tool description
  and REST docs said only "voipbin. ignored", but the sanitizer now also drops the bare reserved
  keys (`flow_id`, `reference_type`, `reference_id`, `reference_activeflow_id`,
  `complete_count`). A customer sending `${flow_id}` would have it silently dropped with no doc
  warning. Added the explicit bare-key list to the tool schema description and a REST
  "Reserved keys" note.
- **L1: "5×64 KB" vs the 256 KB (= 4×64 KB) cap.** Clarified the realistic merged size sits well
  under the cap.
- **M1: counter granularity.** `dropped_reserved` covers both the `voipbin.`-prefix and bare-key
  drops (single outcome bucket is acceptable for Phase-1 observability).

No Critical or High design (non-documentation) issues remained at round 3. The design is
approved for Phase-1 implementation.
