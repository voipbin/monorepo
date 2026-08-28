# VOIP-1405: Global Topic Exchange Publisher Rollout Design

- Date: 2026-08-27
- Ticket: VOIP-1405 (Follow-up A of VOIP-1404)
- Status: Approved (design review: 4 rounds, 2 consecutive approvals)
- Normative base: `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` (rev.7). This document does NOT restate the skeleton; it defines the rollout scope, the per-service subscription-id mapping (normative), the prerequisite fixes, and the decided policy questions. Where this doc and the 1404 design conflict, this doc wins for rollout matters only.

## 1. Scope

### 1.1 Services gaining `WithGlobalTopicPublish()` — 26 services, 53 wiring points

agent(2) ai(3) billing(2) call(2) campaign(2) conference(2) contact(2) conversation(2) customer(3) direct(2) email(2) flow(2) message(2) number(2) outdial(2) pipecat(2) queue(3) registrar(4) route(2) schedule(1) sentinel(1) storage(3) tag(2) talk(1) tts(1) webchat(1)

- Numbers are NotifyHandler construction sites gaining the option per service, AFTER exclusions (talk and tts each have one more site, excluded below); ai-control has 2 instances, customer-control 2, registrar-control 3, storage-control 2, queue-control 2.
- talk: `talk-manager` only. `talk-control` constructs NotifyHandler with `queueEvent=""` (defect, separate ticket) — excluded.
- tts: `tts-manager` only. `tts-control` constructs a NotifyHandler but injects it solely into `pkg/ttshandler`, which has zero publish calls (same dead-dependency fact as §7) — no live publish path.
- registrar-control publishes directly from cmd code (`domain_migrate.go:591`) — all 3 of its constructors get the option (publisher-stream completeness, per 1404 §5.3 rationale).

### 1.2 Exclusions (with reasons — do not re-litigate)

| Class | Services | Reason |
|---|---|---|
| Non-publishers | api-manager, hook-manager, rag-manager, timeline-manager, transfer-manager, kamailio-proxy, rtpengine-proxy, trigger-sender | Zero `PublishEvent*` call sites (transfer-manager holds a dead NotifyHandler dependency — separate cleanup ticket) |
| Publisher, option forbidden | webhook-manager | Publishes only via `PublishEventWithRoutingKey` to the scope-first exchange (1404 §5.2 "untouched"); enabling would triple-publish — contract encoded in `notifyhandler/main.go:214-216` |
| Permanently excluded | voip-asterisk-proxy | ARI raw relay, 1404 §2 Non-Goals |
| Structurally out of scope | bin-common-handler (shared library), bin-dbscheme-manager (Alembic/Python), bin-openapi-manager (spec generator) | No service/publishing code by definition — closes the 40-directory reconciliation: 26+8+1+1+1(transcribe pilot)+3 = 40 |

## 2. Subscription-id Mapping (NORMATIVE — source for every golden-key test)

> **SUPERSEDED IN PART (VOIP-1419, 2026-08-28):** the "JSON top-level `id` fallback" defaults in
> this section became explicit `EventSubscriptionID()` methods returning the same values; the
> fallback itself was deleted and the interface is compile-time mandatory. The ADDRESS mapping
> below (which id each type resolves to) is unchanged and remains normative. See
> `2026-08-28-voip-1419-explicit-subscription-id-design.md`.

Counting basis: Go types receiving `EventSubscriptionID()` (pointer receiver) = Category A 8 + Category B 11 = **19 types**, including **4 new structs** (call whitelist 1 + contact case 3). (Design review R1 moved `CaseNoteDeletedEvent` A→B and added `queuecall` to B; the issue-analysis figure of 18 predates the queuecall addition.)

### 2.1 §4.2 extension — Category B policy (amends 1404 design)

A child resource whose own id is stable but whose natural consumption axis is a parent stream adopts the **parent id as its subscription address**. Trade-off, accepted deliberately: *adopting the parent address forfeits own-id instance subscription for that child.* Grounds: (i) the child id first appears in its `created` event, so own-id pre-binding has no value; (ii) single-item retrieval remains available via RPC; (iii) every real consumption pattern (session/case/conversation following) is parent-axis, and the production webhook path already addresses chat children by parent (`bin-webhook-manager/pkg/webhookhandler/routingkey.go:175` and `createRoutingKeysForChat`, which uses chatID with own-id only as nil-fallback) — a precedent, not merely an analogy.

### 2.2 Category A — own id structurally unusable (override mandatory)

| Service | Type | Address field | Evidence |
|---|---|---|---|
| ai | `*message.IntermediateWebhookMessage` | `AIcallID` | non-persisted streaming fragment, `pkg/messagehandler/event.go:275`, has `Sequence` |
| call | `*dtmf.DTMF` | `CallID` | fresh uuid per event (`pkg/callhandler/digit.go:208`) |
| call | **new struct** `call.OutboundWhitelistRejectedEvent{call_id, customer_id, destination_country}` | `CallID` | today `map[string]interface{}` value at `outgoing_call.go:213` — no top-level id, map cannot satisfy a pointer-receiver assertion |
| contact | **new structs** `kase.CaseTagEvent{case_id, tag_id}`, `kase.CaseContactEvent{case_id, contact_id}` (existing `models/kase` package — `case` is a reserved word; JSON keys are the contract) | `CaseID` | today `map[string]uuid.UUID` values at `case_tag.go:92,130`, `contact_update.go:76` (dynamic 2 event types) — no top-level id, 3 sites / 4 event types converge to 2 structs with byte-identical JSON keys |
| pipecat | `*message.Message`, `*message.MemberSwitchedEvent` | `PipecatcallID` | id-generation rule differs per publish site (random `UUIDCreate()` at `runner.go:521` vs `messageID` at 865/886); `MemberSwitchedEvent` has no top-level id. **Prerequisite: pointer conversion of 6 value publishes** (`runner.go:510,563,576,584,865,886`) |
| tts | `*message.Message` | `StreamingID` | `Message.ID` fixed once at streamer Init (`pkg/streaminghandler/gcp.go:178-184`, aws/elevenlabs analogous); 2nd+ utterances diverge from `streaming.MessageID` (stale); `SayStop`'s `uuid.Nil` write touches only the streaming record (`say.go:28-31,42`, `streaming.go:137`), never the published payload |

### 2.3 Category B — stable own id, parent stream is the address (policy §2.1)

| Service | Type | Address field |
|---|---|---|
| ai | `*message.Message` | `AIcallID` |
| campaign | `*campaigncall.Campaigncall` | `CampaignID` |
| conference | `*conferencecall.Conferencecall` | `ConferenceID` |
| contact | `*casenote.CaseNote` | `CaseID` |
| contact | **new struct** `casenote.CaseNoteDeletedEvent{id, case_id, customer_id}` | `CaseID` | 
| conversation | `*message.Message` | `ConversationID` |
| queue | `*queuecall.Queuecall` | `QueueID` (R1: structural twin of campaigncall/conferencecall — 7 lifecycle events follow the queue axis; consistency with §2.1) |
| schedule | `*execution.Execution` | `ScheduleID` (succeeded/failed share one persisted row) |
| talk | `*message.Message`, `*participant.Participant` | `ChatID` |
| webchat | `*message.Message` | `SessionID` |

**`CaseNoteDeletedEvent` silent-failure warning (R1)**: unlike the other 3 new structs, its map today carries a top-level `"id"` (the note id) — a well-formed-but-wrong address that evades the placeholder metric if the override is omitted (the exact CRITICAL class of 1404 §4.2). Its golden-test row asserting address = `case_id` is **mandatory**, and the pairing with `case_note_created` (CaseNote → CaseID) must land in the same key space.

### 2.4 Default (no override) — everything else

All remaining published types use the JSON top-level `id` fallback. Explicitly decided defaults (do not override):

- **billing.Billing** — own id (address consistency with `billing_updated`; id obtainable from create response)
- **activeflow.Activeflow** — own id (`ReferenceID` can be `uuid.Nil` → would inflate placeholder rate)
- **accesskey.Accesskey** — own id (independent persistent resource)
- **customer.Customer** — own id, and **no override may ever be added to `*Customer`**: `CustomerCreatedEvent` anonymously embeds `*Customer`, so a method would promote to the wrapper; also a nil embed marshals without `id` → placeholder path (golden test case required)
- **sentinel `*corev1.Pod`** — external type, no top-level id → **placeholder by design**. Documented runbook exception: `sentinel_manager_topic_placeholder_total` grows steadily and is expected; instance subscription of pod events is not supported.
- **recording.Recording** — own id, deliberate: download/stop APIs are recording-id keyed and the id returns from the start RPC (pre-bindable); call-axis followers use type-level patterns. (Recorded because it superficially resembles transcript.)
- **direct, route/provider/providercall, tag, chat, speaking, tts streaming, webchat session, email, flow, number, outdial, storage account/file, agent, team, summary, aicall, conference, confbridge, groupcall, trunk, extension, schedule, billing account** — own id (several declare `ID` directly instead of embedding Identity; JSON tag `"id"` is what matters).

## 3. Prerequisite Fixes (in this ticket, before/with wiring)

1. **contact-manager `[]byte` publish** (`pkg/contacthandler/event.go:29`): replace `CreateWebhookEvent()` `[]byte` with `ConvertWebhookMessage()` `*WebhookMessage`. Impact, measured: no runtime consumer breakage (sole consumer timeline-manager stores `string(e.event.Data)` verbatim; contact is not in the peer_event whitelist). The events-table history switches from base64 string to JSON object mid-stream — **intended improvement**, noted in PR body. Without the fix, topic publishes would be all-placeholder.
2. **pipecat value publishes ×6** → pointer (`&msg`) or `newMessageEvent` returns `*message.Message`. Payload bytes unchanged.
3. **map→struct conversions** (call ×1, contact ×4 sites): new structs in §2.2/§2.3 with identical JSON key SETS (field order differs from map marshaling — Go maps marshal key-sorted, structs in declaration order; JSON-semantically equivalent, do not assert byte equality). Payload compatibility contract: fanout consumers observe no semantic change. Note: pipecat is the ONLY service publishing value types — every other service already passes pointers; no other pointer conversions exist.
4. Contact case event types get named constants (today string literals) — required by golden tests.

## 4. Golden-Key Tests (per service, template = transcribe pilot)

- One table per service covering **every actually-published event type**. Dead constants excluded — 15 known, service-qualified to avoid excluding live twins: outdial `outdialtarget_created/updated/deleted`, queue `queue_created`, sentinel `pod_added`, storage `file_updated`, tts `streaming_finished`, `streaming_play_started`, `streaming_play_finished`, conversation `conversation_deleted`, campaign `outplan_created/updated/deleted`, billing `billing_deleted`, billing `account_deleted`. **Caution**: conversation `account_deleted` and storage `Account_deleted` are LIVE events — the dead one is billing's only. Likewise tts: `message_play_started`/`message_play_finished` are LIVE (all 3 vendor backends) — only the `streaming_*` trio is dead.
- Dynamic eventType sites must enumerate all branches: call `mapEvt` 5 types (`db.go:288,298`), contact contact_update 2, number dbUpdate 2, schedule dispatch 2.
- Resolution helper mirrors notifyhandler order (interface assertion first, JSON fallback second, override-authoritative) — copy from `bin-transcribe-manager/models/transcribe/routingkey_golden_test.go`.
- Mandatory special cases: storage `Account_*` uppercase normalization; call dot-type `call.outbound_whitelist_rejected`; customer `CustomerCreatedEvent` wrapper (id promotion + nil-embed placeholder); sentinel pod placeholder; `CaseNoteDeletedEvent` address (see §2.3 warning); webchat resource collapse (both event types → resource `webchat`, same session address — one pattern follows the whole session).
- Address-convergence notes (consumer contract, mirror 1404 §4.2's three-namespace statement): **tts** — `message_*` (StreamingID) and `streaming_*` (own id = streaming id) converge on one address: `tts-manager.streaming.<id>.#` + `tts-manager.message.<id>.#` follow a session. **contact** — casenote/case-tag/case-contact events converge on the case axis: `contact-manager.case.<case-id>.#` covers the case lifecycle. **pipecat** — pipecatcall/message/team namespaces share the pipecatcall id (three patterns).
- Compile-time assertions `var _ eventtopic.SubscriptionIdentifier = (*T)(nil)` for all 19 override types.

## 5. Rollout, Verification, Observability

- **Wiring**: add `notifyhandler.WithGlobalTopicPublish()` at each of the 53 sites. No bin-common-handler changes (assert with `git diff --stat`).
- **Verification per service**: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` (+ `go test -cover` on changed packages). **Local vendor/ is stale (pre-1404) in every service — `go mod vendor` is mandatory before tests.** Cross-module compile sweep over all 39 go.mod modules at the end.
- **Docs sync per service**: `docs/architecture.md` (cmd/main.go hook rule) + `docs/domain.md` where models change. Reference doc `docs/reference/rabbitmq-queues-reference.md`: extend the key-schema section with the per-service mapping summary (Category A/B tables) and the sentinel placeholder exception.
- **Deployment**: services deploy independently (existing per-service pipelines); no coordinated cutover needed — option-on is additive. Suggested wave order for observation: low-frequency group first (tag/direct/email/number/...), call-manager and pipecat (highest volume) last. Post-deploy checks per 1404 §7 runbook: `<ns>_topic_publish_total{result="ok"}` growth, `<ns>_topic_placeholder_total` ~0 — sentinel exception is NOT "ignore the metric": for sentinel the healthy invariant is `placeholder_total ≈ topic_publish_total{result="ok"}` (100% placeholder by design), which still detects publish regressions. Broker channel churn comparison.
- **Rollback**: per service, single-line option removal. Note: option rollback does NOT revert the §3.1 contact payload normalization (base64→JSON in stored history) — that is an intended, independent change and stays.

## 6. Decided Questions

(Of the issue analysis's 7 decision points, 3 — billing/activeflow/accesskey defaults — plus the sentinel placeholder are recorded normatively in §2.4; the remaining 4 are here.)

1. delay>0 topic semantics: **closed as not-applicable** — no public API produces delay>0 (re-verified in 1404 review); the guard stays defensive.
2. CLI (*-control) inclusion: all instances with a live publish path (see §1.1); publish-less controls excluded.
3. conversation-manager inclusion: **included**. `account.Secret/Token` ride in internal event payloads today (fanout) and continue to on the topic exchange; broker access already requires internal credentials, so the **audience** is unchanged — but the **discovery barrier drops**: today a reader must know and bind conversation-manager's specific fanout exchange, whereas a single `#` binding on `bin-manager.event` surfaces these payloads alongside everything else. Accepted with eyes open; a **priority follow-up ticket** is registered to strip secrets from internal event payloads.
4. Single PR: yes (policy default), exceptional size acknowledged (~100-150 files). Commits structured per service group; PR body carries a per-service summary table.

## 7. Follow-up tickets to register at PR time

- talk-control `queueEvent=""` defect (verify broker behavior for `TopicCreate("")`, then fix; note `reqHandler` is also nil there). After the fix, add `WithGlobalTopicPublish()` there too — otherwise talk-control publishes fanout-only, violating the same publisher-stream completeness principle that §1.1 applies to registrar-control
- Dead NotifyHandler dependencies: transfer-manager (2 cmds) + tts `pkg/ttshandler`
- conversation-manager internal event payload secret stripping (priority)
- Event-emission gaps (flow `updateNextAction`, email `UpdateProviderReferenceID`, conversation_deleted never published) — stream-completeness audit
