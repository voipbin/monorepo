# VOIP-1404: Global Topic Exchange (`bin-manager.event`) Design

- Date: 2026-08-27
- Ticket: VOIP-1404
- Status: Approved (design review: 5 rounds, 2 consecutive approvals)
- rev.5: §4.1 publisher normalization amended to match implementation (code-review round 1); rev.6: §5.2 hasOverride threading + option-gated/nil-guarded resolution amended to match implementation (code-review round 2); rev.7: §4.1/§4.2/§4.3 subscription-id 64-byte length cap and IsPlaceholderSubscriptionID amended to match implementation (code-review round 3)
- Prerequisite reading: issue analysis (VOIP-1404 ticket description), `docs/plans/2026-07-12-voip-1233-rabbitmq-ack-after-process-design.md`, `bin-webhook-manager/pkg/webhookhandler/routingkey.go` (VOIP-1258/VOIP-1296 precedent)

## 1. Problem

Service-to-service events are published to per-service fanout exchanges (`bin-manager.<service>.event`). Every subscriber receives the publisher's entire event stream and filters in-process via `switch m.Publisher/m.Type`, discarding most messages. Measured: 21 subscribing services process 70 publisher/type combinations out of ~190-213 event types system-wide (count varies by whether all `EventType*` constants or only published ones are included).

Selective-subscription demand is growing. Canonical example (future feature, not in this ticket): ai-manager wants real-time transcription for one specific call — which concretely means the `transcribe_speech_interim` stream plus `transcript_created` finals. Today that requires subscribing to transcribe-manager's whole stream and comparing IDs on every message.

Broker-level filtering requires a topic exchange with meaningful routing keys. This ticket builds only the skeleton: topology, publish path, subscription capability, and one pilot publisher.

## 2. Goals / Non-Goals

### Goals
1. Global topic exchange `bin-manager.event` (single, durable, topic kind).
2. Routing key schema `<publisher>.<resource>.<subscription-id>.<action>`, one event = one key = one publish.
3. Dual publish (existing fanout + new topic) in `bin-common-handler/pkg/notifyhandler`, opt-in per service.
4. Subscription capability: key/pattern builder utilities usable with the existing `QueueBind`/`QueueUnbind`.
5. Pilot wiring: transcribe-manager (both `cmd/transcribe-manager` and `cmd/transcribe-control`) publishes to the topic exchange.
6. Observability sufficient to detect silent publish failure (the 2026-07-14 VOIP-1258 lesson).

### Non-Goals (follow-up tickets)
- Migrating the other ~27 publishers, migrating any consumer, fanout removal (cutover). Follow-up tickets will be registered at PR time to avoid a permanent dual-publish state (VOIP-1296 precedent).
- Instance-subscription convenience API (`Follow`/refcount), per-pod volatile queue helpers.
- Client-generated resource IDs (requires per-API request schema changes; `V1DataTranscribesPost` has no ID field today).
- Consumer-side state-repair flows.
- `asterisk.all.event` (ARI stream) stays out of this topology **permanently**: voip-asterisk-proxy is the sole `PublishEventRaw` caller (`pkg/eventhandler/ari_handler.go:76`, every ARI frame as `ari_event`) and the platform's highest-volume publisher; its payloads have no top-level `id`, so topic keys would be all-placeholder noise. It is explicitly excluded from Follow-up A's "remaining publishers" scope.
- Per-publish channel reuse in `rabbitmqhandler.publishExchange` (pre-existing library behavior, see §5.5; tracked as a known optimization candidate, not changed here).

## 3. Topology

```
bin-manager.event          exchange, kind=topic, durable=true, autoDelete=false
```

- Precedent for a 2-segment global exchange: `bin-manager.delay`.
- **Declaration invariant**: both publishers and subscribers declare the exchange at startup via the shared helper only (`sockhandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic")`; durable=true/autoDelete=false are hardcoded in `rabbitmqhandler/topic.go`). Declaration is idempotent iff parameters match exactly; a mismatched redeclare closes the channel with 406 PRECONDITION_FAILED. Hand-rolled declarations are forbidden; code review enforces.
- Subscriber-side declaration is required for start-order independence (a subscriber that boots before any publisher must not fail its `QueueBind`).

The existing scope-first client-facing exchange `bin-manager.webhook-manager.event.topic` (`customer_id.<uuid>....` keys) is a separate channel with a different audience (customer/agent-scoped WebSocket/webhook delivery). The new exchange carries internal service-to-service events; internal services operate cross-tenant, so the key has no customer segment by design. Both coexist permanently. Deliverable: a new "Exchanges" section in `docs/reference/rabbitmq-queues-reference.md` documenting both roles (that doc currently covers queues only).

## 4. Routing Key Schema

```
<publisher>.<resource>.<subscription-id>.<action>

call-manager.call.4a539340-a1bc-11f1-92ef-60452e5e40a2.created
transcribe-manager.transcript.9f01c3d2-....created          # subscription-id = TranscribeID
transcribe-manager.transcribe.9f01c3d2-....speech_interim   # subscription-id = TranscribeID
```

### 4.1 Segment derivation

| Segment | Source | Rule |
|---|---|---|
| publisher | `notifyhandler.publisher` (service name, set at construction) | normalized (shared `normalizeSegment` — key↔pattern match invariant; no-op for all real service names) |
| resource | normalized `EventType` | first segment of `strings.SplitN(normalized, "_", 2)` |
| subscription-id | event data (§4.2) | UUID string, or `-` placeholder |
| action | normalized `EventType` | remainder after the first `_`; if no `_` exists, the whole normalized type is the action and resource is `-` |

**Normalization (applies before splitting)**: lowercase the event type and replace every `.` with `_`. After segment computation, any residual AMQP-significant character (`.`, `*`, `#`) or empty segment is replaced with `_` / `-` respectively. This handles real outliers found in the codebase:

- `call.outbound_whitelist_rejected` (bin-call-manager) → normalized `call_outbound_whitelist_rejected` → resource `call`, action `outbound_whitelist_rejected` (4 segments preserved; groups under the `call` resource with its siblings).
- `Account_created` (bin-storage-manager) → normalized `account_created`.

The split is otherwise purely mechanical, mirroring `webhookhandler.routingkey.go`. Multi-underscore types (e.g. `customer_balance_updated`) split as `customer` / `balance_updated`. This is accepted: what matters for binding is that generated keys are deterministic and stable, not that segments are semantically perfect. The reference doc lists the generated shape per event family; per-publisher golden-key tests (§8) pin the actual output.

Constraint check: max routing key length is 255 bytes; worst case here is ~119 bytes (publisher ≤20 + subscription-id ≤64 (§4.2 length cap; UUIDs are 36) + normalized type ≤32 + 3 dots).

### 4.2 subscription-id: "subscription address", not "own id"

The third segment answers "by which ID will subscribers address this stream?".

- Default: the resource's own top-level `id`.
- Stream-child resources override with their parent stream id via the opt-in interface below. A per-utterance or per-result id that is newly generated for every event is not an address — nobody can bind to it in advance. **A non-Nil-but-meaningless id is worse than no id**, because it produces well-formed keys that match nothing and evades the placeholder metric; the defense against this class is the per-publisher golden-key test plus the mapping table below, not runtime detection.

Mechanism (compile-time, no reflection):

```go
// bin-common-handler/models/eventtopic
// Opt-in override. Checked by type assertion on the event data before marshaling.
// Implement with a POINTER receiver: event data is passed as a pointer
// (e.g. *transcript.Transcript) and the assertion must match the dynamic type.
type SubscriptionIdentifier interface {
    EventSubscriptionID() string
}
```

- If the data implements `SubscriptionIdentifier`, use it.
- Else, marshal-then-extract the top-level `"id"` JSON field (this is the same marshal already performed for the fanout publish; the extraction reuses those bytes, one extra small unmarshal into a `struct{ ID string }`).
- If the result is empty, `uuid.Nil` ("00000000-..."), **or longer than 64 bytes**: use placeholder `-`. Type-level bindings (`<publisher>.<resource>.#`) still match; instance bindings never match, which is correct because no valid address exists. The placeholder rate is metered (§7) so absent-id drift is visible. (`uuid.Nil` → placeholder specifically makes the VOIP-1258 all-Nil failure mode observable. The 64-byte cap protects the AMQP 255-byte routing-key limit against a future publisher whose address is not a UUID — an oversized address degrades to a placeholder key rather than a rejected publish, i.e. a lost event. `PatternInstance` applies the same cap, so keys and instance patterns stay consistent; the exported predicate `IsPlaceholderSubscriptionID` is the single owner of all three placeholder rules and is what the publish path uses to meter the placeholder counter. Subscribers must know: an address over 64 bytes cannot be instance-subscribed.)

#### Pilot mapping table (exhaustive for transcribe-manager)

| EventType(s) | Data type | subscription-id | How |
|---|---|---|---|
| `transcribe_created`, `transcribe_progressing`, `transcribe_done`, `transcribe_deleted` | `*transcribe.Transcribe` | own `ID` (= transcribe-id) | default JSON `id` fallback |
| `transcribe_speech_interim/started/ended` | `*streaming.Speech` | `TranscribeID` | `SubscriptionIdentifier` override (own `ID` is random per event — must not be the address) |
| `transcript_created` (and other `transcript_*`) | `*transcript.Transcript` | `TranscribeID` | `SubscriptionIdentifier` override |
| `streaming_started/stopped` | `*streaming.Streaming` | `TranscribeID` | `SubscriptionIdentifier` override (own `ID` is the streaming-id, a stable-but-wrong address for session followers; `Streaming` carries `TranscribeID`) |

Result: every transcribe-manager event about one transcription session carries the same address (transcribe-id) across three resource namespaces. A consumer following one session binds three patterns: `transcribe-manager.transcribe.<id>.#`, `transcribe-manager.transcript.<id>.#`, `transcribe-manager.streaming.<id>.#` (the last only if it cares about streaming lifecycle).

Follow-up A (other publishers) must produce the same table per service before wiring; the golden-key test template enforces it.

### 4.3 Pattern builders

```go
// bin-common-handler/models/eventtopic (pure functions, unit-tested)
func RoutingKey(publisher, eventType, subscriptionID string) string
func PatternAll(publisher string) string                     // "<p>.#"
func PatternResource(publisher, resource string) string      // "<p>.<r>.#"
func PatternInstance(publisher, resource, id string) string  // "<p>.<r>.<id>.#"
func PatternAction(publisher, resource, action string) string // "<p>.<r>.*.<action>"
func IsPlaceholderSubscriptionID(id string) bool // single owner of the placeholder rules (empty / uuid.Nil / >64 bytes)
```

Consumers combine these with the existing `sockhandler.QueueBind/QueueUnbind`. No new consumer framework in this ticket.

Shared-binding caveat (documented in package doc): a broker binding is shared by all logical subscribers using the same queue+pattern; one `QueueUnbind` severs all of them, and `QueueBind` is idempotent in the reconnect-tracking list. Multi-subscriber users must keep refcount discipline (0↔1 transition only), as `bin-api-manager/pkg/websockhandler/scoperefcount.go` does. A shared refcount helper is a follow-up.

**Admission-rule note**: `models/eventtopic` lands in `bin-common-handler` although only one service uses it at pilot time. Justification: it is internal plumbing of `notifyhandler` itself (a shared-library component); the 3+-services rule targets service-level utilities, and every publisher adopts this package in Follow-up A.

## 5. Publish Path

### 5.1 notifyhandler changes

Both constructors gain a backward-compatible variadic option parameter:

```go
func NewNotifyHandler(sock, req, queueNotify, publisher, opts ...Option) NotifyHandler
func NewNotifyHandlerForExistingExchange(sock, req, queueNotify, publisher, opts ...Option) NotifyHandler

// pilot service only; default is off everywhere:
notifyhandler.WithGlobalTopicPublish()
```

Why an option rather than a third named constructor (the `NewNotifyHandlerForExistingExchange` precedent): the two axes (existing-exchange × global-topic) would multiply named constructors (4 combinations); a single option composes with both existing constructors without breaking any call site (variadic addition is source-compatible). This introduces the constructor-level functional-option pattern to this package (nearest repo precedent, `bin-ai-manager/pkg/messagehandler`, uses method-level options; constructor-level is new here); the deviation is deliberate and documented here.

The option sets a `notifyHandler` struct field (`topicEnabled`), symmetric with the `topicDisabled` degradation flag (§5.2) — both are per-instance state, never package globals.

Interaction contract: `WithGlobalTopicPublish` on the existing-exchange constructor is valid but **must not be enabled for webhook-manager's scope-first instance** in this ticket (it would triple-publish webhook events); nothing enables it there.

When enabled, the constructor declares `bin-manager.event` via the shared helper (§3).

### 5.2 Hook insertion point and API coverage

The private common path `publishEvent(eventType, dataType string, data json.RawMessage, timeout, delay int)` receives **already-marshaled bytes**, so the `SubscriptionIdentifier` type assertion cannot happen there. Resolution — split responsibilities:

1. **Subscription-id resolution happens in the public methods that still hold `data interface{}`** (`PublishEvent`; `PublishWebhookEvent` delegates via `go h.PublishEvent(...)` unchanged). Resolution distinguishes *override exists* from *override value*: the pair is threaded into `publishEvent` as new parameters: `publishEvent(eventType, dataType string, data json.RawMessage, timeout, delay int, subscriptionID string, hasOverride bool)`. The resolution is gated on the option being enabled (option off ⇒ no type assertion at all), and a typed-nil pointer never has its method called (reflect nil-guard) — it resolves as no-override.
2. **The topic publish itself happens inside `publishEvent`**, after the fanout publish attempt, guarded by the option flag and `delay == 0`. The JSON `"id"` fallback on the marshaled bytes (§4.2) runs **only when no override exists** (`hasOverride == false`); an existing override is authoritative — if its value is empty or `uuid.Nil`, the key goes straight to the `-` placeholder (never the fallback). Fallback results that are empty/Nil likewise become `-`.

| Public API | Subscription-id source | Topic publish |
|---|---|---|
| `PublishEvent` | type assertion in `PublishEvent` (option-gated, nil-guarded); no override → JSON fallback in `publishEvent` | **yes** |
| `PublishWebhookEvent` | via `PublishEvent` — no separate handling | **yes** (via PublishEvent) |
| `PublishEventRaw` | passes `""` — `data []byte` cannot satisfy the assertion; JSON fallback only. Contract documented in code. Sole caller today: voip-asterisk-proxy's ARI relay (`ari_handler.go:76`), which is permanently excluded from topic opt-in (§2 Non-Goals) — so this row is future-proofing, not an active path. | **yes** (when option enabled) |
| delay>0 path (`publishDelayedEvent`) | n/a | **no** — note: no public API produces delay>0 today (both `publishEvent` call sites pass literal `0`); the guard is defensive. Delayed-event topic semantics are deferred to Follow-up A. The §8 test for this branch targets the private function; rationale is this defensiveness. |
| `PublishEventWithRoutingKey` (webhook scope-first path) | n/a — separate path | **no** — untouched. |

**Fanout-failure interaction**: `publishEvent` today early-returns when the fanout publish fails. That behavior is **kept**: fanout failure ⇒ topic publish is skipped. Rationale: during dual publish the fanout path is the system of record; delivering an event on topic that fanout consumers never saw would create divergent state. (The independence requirement is one-directional: topic failure must not affect fanout, per the failure-isolation rule later in this section.)

**Exchange-declare failure contract**: unlike the existing constructor behavior (`NewNotifyHandler` returns `nil` when the fanout `TopicCreate` fails), a failure to declare `bin-manager.event` must **not** nil out the handler — the topic exchange is strictly secondary during dual publish. Contract, fully specified:

1. **Metric registration is unconditional**: the two new counters (§7) are registered inside `initPrometheus` regardless of the option, alongside the existing counters, under the existing `initPrometheusDone[namespace]` guard. This avoids both permanent non-registration and duplicate-registration panics, and guarantees the counters are non-nil before any use.
2. **Constructor ordering**: the `bin-manager.event` declare runs **after** `initPrometheus`. In `NewNotifyHandler` that also means after the existing fanout `TopicCreate`; `NewNotifyHandlerForExistingExchange` performs no fanout declare, so there the rule is simply "after `initPrometheus`".
3. **On declare failure**: log Error and set `topicDisabled` — a **`notifyHandler` struct field**, not a package global (multi-instance processes like webhook-manager must not share the outcome across instances; set-in-constructor keeps it race-free against `PublishWebhookEvent`'s goroutine fan-out). The constructor returns a working handler with the fanout path fully alive. No counter is incremented at declare time (no `type` label exists yet). Every subsequently *suppressed* topic publish increments `<ns>_topic_publish_total{type, result="error"}` — safe because it runs post-`initPrometheus` with a real event type.
4. **Publish primitive**: the topic publish calls `h.sockHandler.EventPublish(string(commonoutline.QueueNameEvent), routingKey, evt)` directly — it must **not** reuse `publishDirectEvent`/`publishDirectEventWithKey`, both of which observe `promNotifyProcessTime` and would violate the metrics-isolation rule mechanically.

This degradation is safe at the AMQP level because `rabbitmqhandler` opens a dedicated channel per exchange declare, so a 406 PRECONDITION_FAILED closes only that channel and cannot poison the fanout path. Silent swallowing is not acceptable (VOIP-1258 lesson) — the Error log plus `{result="error"}` growth plus the absence of `{result="ok"}` growth are the detection signals, and the §7 runbook checks exactly that.

- Fanout publish executes first, exactly as today (order and error semantics unchanged).
- **Failure isolation**: topic publish failure is logged + counted (§7) and never propagated to the caller, never affects the fanout publish. During dual publish the fanout path is the system of record.
- The topic publish reuses the marshaled payload; the `sock.Event` on the topic exchange is byte-identical to the fanout payload (same `Type`, `Publisher`, `DataType`, `Data`). Consumers migrating later reuse their existing decode path.
- **Metrics isolation**: the topic path must not touch `promNotifyTotal` **or `promNotifyProcessTime`** (both are observed by the existing direct-publish helpers; reusing them would double-count). The topic publish is a separate code path with its own counters only (§7).

### 5.3 Pilot: transcribe-manager

- `cmd/transcribe-manager/main.go` **and** `cmd/transcribe-control/main.go`: both construct their own NotifyHandler and publish to the same logical stream; both enable `WithGlobalTopicPublish()` so the "transcribe-manager events exist on the topic exchange" contract holds for consumers regardless of which process published.
- `models/streaming.Speech`: implement `EventSubscriptionID()` (pointer receiver) returning `TranscribeID.String()`.
- `models/streaming.Streaming`: implement `EventSubscriptionID()` (pointer receiver) returning `TranscribeID.String()` (own `ID` is the streaming-id — a stable-but-wrong address, per §4.2).
- `models/transcript.Transcript`: implement `EventSubscriptionID()` (pointer receiver) returning `TranscribeID.String()`.
- `models/transcribe.Transcribe`: default `id` extraction suffices (no override).
- Known pre-existing bug on this path: `pkg/transcripthandler/db.go:33` publishes `EventTypeTranscriptCreated` on the delete path (should be Deleted). Out of scope; separate Jira ticket to be filed.

### 5.4 bind-after-start race (consumer contract, no code this ticket)

`transcribe start` generates the transcribe-id inside `startLive` and begins streaming before returning, so a consumer binding on the RPC response may miss the first event(s); unroutable topic messages are silently dropped (mandatory=false). Resolution candidates, decided per consumer at adoption time:
- (a) **Recommended default**: after bind, call `TranscribeV1TranscriptList` (exists today) once to repair initial state.
- (b) Client-generated id via API change (bind-before-start, lossless) — future option.
- (c) Documented initial-loss tolerance (acceptable for interim speech results, which are ephemeral by nature).

This contract is recorded in the reference doc; the skeleton is independent of the choice.

### 5.5 Load assessment (pilot)

The highest-frequency pilot event is `transcribe_speech_interim` (published per interim STT result, `streaminghandler/result.go` — one or more per utterance), not `transcript_created` (finals only). Cost model of dual publish:

- `rabbitmqhandler.publishExchange` opens and closes an AMQP channel **per publish** (pre-existing behavior, flagged as an open question in the VOIP-1233 design §8-5). Dual publish therefore doubles channel open/close churn for transcribe-manager events, not just broker ingress.
- Routing itself is near-no-op while no bindings exist.
- Mitigations in this ticket: (1) measure before/after — current per-type publish rate is available from `transcribe_manager_notify_total{type}` in Prometheus (metric namespace = service name with `-`→`_` via `commonoutline.GetMetricNameSpace`; the bare name is `notify_total`); capture a baseline during design sign-off and compare post-deploy channel churn via RabbitMQ `channel_created` stats; (2) rollback is a single-line option removal per cmd; (3) channel reuse in `publishExchange` is the structural fix and stays a tracked library follow-up (Non-Goals) — if post-deploy churn is problematic, that follow-up is the remedy, not schema changes.
- Lock-hold interaction: `streaminghandler`'s Create/Delete call `PublishEvent` synchronously while holding the handler mutex, so dual publish doubles the channel open/close performed under that lock. Accepted: these are per-streaming-leg lifecycle events (low frequency, unlike the per-interim-result stream above), and the added hold time is one channel round-trip. Observation needs no separate instrument — it is already covered by the channel-churn comparison above.
- Scale context: single-broker bm-nyc-01, transcribe traffic is bounded by concurrent transcribing calls; expected absolute rates are well within RabbitMQ channel-churn capacity. The measurement exists to verify, not to gate.

## 6. Constants

`bin-common-handler/models/outline/queuename.go`:

```go
// QueueNameEvent is the global topic exchange for service-to-service events (VOIP-1404).
QueueNameEvent QueueName = "bin-manager.event"
```

(`QueueName` is the existing conflated name type used for exchanges as well, per `QueueNameDelay` precedent.)

## 7. Observability

VOIP-1258 shipped a topic publisher that silently published nothing for a month because fanout kept working. Countermeasures:

1. **New metrics in notifyhandler** (existing `promNotifyTotal{type}` and `promNotifyProcessTime{type}` are left strictly untouched). Names below use `<ns>` = the per-service metric namespace (`commonoutline.GetMetricNameSpace(publisher)`, e.g. `transcribe_manager`), matching how the existing `notify_total` becomes `transcribe_manager_notify_total`:
   - `<ns>_topic_publish_total{type, result="ok|error"}`
   - `<ns>_topic_placeholder_total{type}` — subscription-id fell back to `-`
   - The new counters deliberately follow the existing package-level-global + `initPrometheusDone[namespace]` pattern of `promNotifyTotal`; the known multi-namespace-per-process limitation of that pattern is pre-existing, untriggered today (all multi-handler cmds use a single publisher constant), and out of scope.
   - Detection coverage note: the placeholder metric catches *absent* ids; *wrong-but-present* ids (the CRITICAL class found in review: per-event random UUIDs) are caught at build time by golden-key tests (§8), not at runtime.
2. **Post-deploy verification procedure** (runbook section in the PR):
   - RabbitMQ management: confirm `bin-manager.event` exists with kind=topic; confirm non-zero publish-in rate on the exchange while a call with transcribe is active.
   - Temporary verification binding: bind a scratch queue with `transcribe-manager.#`, observe live messages, inspect one message's routing key shape (segment count = 4, third segment = the session's transcribe-id), then delete the queue.
   - Prometheus: `transcribe_manager_topic_publish_total{result="ok"}` increasing; `transcribe_manager_topic_placeholder_total` ~0 for transcript/speech/streaming events; channel churn comparison per §5.5.

## 8. Testing

- `models/eventtopic`: table-driven unit tests for `RoutingKey` and every pattern builder — normal, no-underscore type, multi-underscore, **dot-containing type (`call.outbound_whitelist_rejected`)**, **uppercase type (`Account_created`)**, empty type, nil-UUID → placeholder, empty id → placeholder, oversized id (>64 bytes) → placeholder (64 bytes passes), `SubscriptionIdentifier` override (pointer receiver, asserted against pointer dynamic type), fallback `"id"` JSON extraction, non-JSON data.
- **Golden routing-key test (pilot)**: one table covering *every* event type transcribe-manager publishes (per §4.2 mapping table — transcribe/speech/transcript/streaming families) asserting the exact generated key shape. This is the primary defense against the "wrong id space under a resource namespace" defect class; the table doubles as the Follow-up A template. Note: the golden table pins *current* behavior, including the known `db.go:33` bug (delete path publishes `transcript_created`); the entry must be updated together with that bug's fix ticket.
- `notifyhandler` internal: subscription-id resolution threading — `PublishEvent` resolves via assertion; `publishEvent` receives `""` from `PublishEventRaw` and performs JSON fallback; declare-failure degradation (topic disabled, fanout alive, handler non-nil).
- `notifyhandler`: gomock `SockHandler` tests — dual publish ordering; topic failure isolation (fanout still succeeds, no error to caller); option off ⇒ zero topic publishes; `PublishEventRaw` ⇒ JSON-fallback key; delay>0 ⇒ no topic publish; **`promNotifyTotal`/`promNotifyProcessTime` unchanged by the topic path** (assert via metric read or by construction — topic path calls neither helper); new counters increment.
- `bin-transcribe-manager`: `Speech.EventSubscriptionID()`, `Streaming.EventSubscriptionID()`, and `Transcript.EventSubscriptionID()` unit tests; compile-time assertions `var _ eventtopic.SubscriptionIdentifier = (*streaming.Speech)(nil)`, `(*streaming.Streaming)(nil)`, `(*transcript.Transcript)(nil)`; existing publish-path tests extended to assert the option wiring in both cmds.
- Coverage target: ≥80% on new/changed files.

## 9. Migration Plan (recorded here, executed in follow-ups)

1. **This ticket**: skeleton + transcribe-manager dual publish (both cmds).
2. Follow-up A: remaining publishers opt in — each must produce a §4.2-style mapping table + golden-key test before wiring; delayed-event topic semantics decided here.
3. Follow-up B: consumers migrate subscription from fanout to topic patterns (per service; pairs naturally with the planned common subscribe framework / VOIP-1233 error-propagation follow-up).
4. Follow-up C: cutover — remove fanout publish and per-service fanout exchanges (VOIP-1296 precedent). Registered as a ticket immediately so dual publish cannot become permanent by default.

## 10. Rejected Alternatives (from issue analysis)

- Per-service topic exchanges — subscriber wiring complexity, ~30× migration surface, no cross-publisher patterns.
- Multiple routing keys per event — doubles publish volume and duplicates delivery to `#` subscribers.
- Namespace segment in the key — exchange name already namespaces; constant segments carry no routing selectivity.
- `.topic` name suffix — collision-avoidance artifact of VOIP-1258, unnecessary for a fresh name; type is a declared property, not a name.
