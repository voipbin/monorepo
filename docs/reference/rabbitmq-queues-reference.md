# RabbitMQ Queue Reference

> **Source of Truth:** `bin-common-handler/models/outline/queuename.go`

## Overview

All services in the monorepo communicate via RabbitMQ using a consistent queue naming pattern. This document provides a complete reference for all queue names and their purposes.

## Queue Naming Pattern

Every service follows this naming convention:

```
bin-manager.<service-name>.event       # Service publishes events here
bin-manager.<service-name>.request     # Service receives RPC requests here
bin-manager.<service-name>.subscribe   # Service subscribes to events here
```

### Special Queues

| Queue | Purpose |
|-------|---------|
| `bin-manager.delay` | Delayed message delivery (scheduled tasks) |
| `bin-manager.event` | Global topic exchange for service-to-service events — see [Exchanges](#exchanges) |
| `asterisk.all.event` | Asterisk ARI events (channel/bridge state changes) |

## Complete Queue Reference

> **VOIP-1407 note.** The `*Event` constants below still name each service's per-service fanout exchange (`bin-manager.<service>.event`), and the constants themselves are unchanged. But as of VOIP-1407, the great majority of these exchanges no longer receive any publishes — every topic-enabled publisher now publishes exclusively to the global topic exchange `bin-manager.event` (see [Exchanges](#exchanges) below), and the 28 real per-service fanout exchanges left orphaned by that cutover are scheduled for one-time broker-side deletion (see the exchange deletion runbook under [Exchanges](#exchanges)). The `*Request`/`*Subscribe` queue names in this section are unaffected by any of this. `asterisk.all.event` (not a `QueueName*Event` constant — it is `voip-asterisk-proxy`'s dedicated exchange) is the one fanout exchange that is NOT part of this cutover and is not scheduled for deletion; see the asterisk exception below.

### AI Manager
```go
QueueNameAIEvent     = "bin-manager.ai-manager.event"
QueueNameAIRequest   = "bin-manager.ai-manager.request"
QueueNameAISubscribe = "bin-manager.ai-manager.subscribe"
```

### Agent Manager
```go
QueueNameAgentEvent     = "bin-manager.agent-manager.event"
QueueNameAgentRequest   = "bin-manager.agent-manager.request"
QueueNameAgentSubscribe = "bin-manager.agent-manager.subscribe"
```

### API Manager
```go
QueueNameAPIEvent     = "bin-manager.api-manager.event"
QueueNameAPIRequest   = "bin-manager.api-manager.request"
QueueNameAPISubscribe = "bin-manager.api-manager.subscribe"
```

### Billing Manager
```go
QueueNameBillingEvent     = "bin-manager.billing-manager.event"
QueueNameBillingRequest   = "bin-manager.billing-manager.request"
QueueNameBillingSubscribe = "bin-manager.billing-manager.subscribe"
```

### Call Manager
```go
QueueNameCallEvent     = "bin-manager.call-manager.event"
QueueNameCallRequest   = "bin-manager.call-manager.request"
QueueNameCallSubscribe = "bin-manager.call-manager.subscribe"
```

### Campaign Manager
```go
QueueNameCampaignEvent     = "bin-manager.campaign-manager.event"
QueueNameCampaignRequest   = "bin-manager.campaign-manager.request"
QueueNameCampaignSubscribe = "bin-manager.campaign-manager.subscribe"
```

### Talk Manager
```go
QueueNameTalkEvent     = "bin-manager.talk-manager.event"
QueueNameTalkRequest   = "bin-manager.talk-manager.request"
QueueNameTalkSubscribe = "bin-manager.talk-manager.subscribe"
```

### Conference Manager
```go
QueueNameConferenceEvent     = "bin-manager.conference-manager.event"
QueueNameConferenceRequest   = "bin-manager.conference-manager.request"
QueueNameConferenceSubscribe = "bin-manager.conference-manager.subscribe"
```

### Conversation Manager
```go
QueueNameConversationEvent     = "bin-manager.conversation-manager.event"
QueueNameConversationRequest   = "bin-manager.conversation-manager.request"
QueueNameConversationSubscribe = "bin-manager.conversation-manager.subscribe"
```

### Customer Manager
```go
QueueNameCustomerEvent     = "bin-manager.customer-manager.event"
QueueNameCustomerRequest   = "bin-manager.customer-manager.request"
QueueNameCustomerSubscribe = "bin-manager.customer-manager.subscribe"
```

### Email Manager
```go
QueueNameEmailEvent     = "bin-manager.email-manager.event"
QueueNameEmailRequest   = "bin-manager.email-manager.request"
QueueNameEmailSubscribe = "bin-manager.email-manager.subscribe"
```

### Flow Manager
```go
QueueNameFlowEvent     = "bin-manager.flow-manager.event"
QueueNameFlowRequest   = "bin-manager.flow-manager.request"
QueueNameFlowSubscribe = "bin-manager.flow-manager.subscribe"
```

### Message Manager
```go
QueueNameMessageEvent     = "bin-manager.message-manager.event"
QueueNameMessageRequest   = "bin-manager.message-manager.request"
QueueNameMessageSubscribe = "bin-manager.message-manager.subscribe"
```

### Number Manager
```go
QueueNameNumberEvent     = "bin-manager.number-manager.event"
QueueNameNumberRequest   = "bin-manager.number-manager.request"
QueueNameNumberSubscribe = "bin-manager.number-manager.subscribe"
```

### Outdial Manager
```go
QueueNameOutdialEvent     = "bin-manager.outdial-manager.event"
QueueNameOutdialRequest   = "bin-manager.outdial-manager.request"
QueueNameOutdialSubscribe = "bin-manager.outdial-manager.subscribe"
```

### Pipecat Manager
```go
QueueNamePipecatEvent     = "bin-manager.pipecat-manager.event"
QueueNamePipecatRequest   = "bin-manager.pipecat-manager.request"
QueueNamePipecatSubscribe = "bin-manager.pipecat-manager.subscribe"
```

### Queue Manager
```go
QueueNameQueueEvent     = "bin-manager.queue-manager.event"
QueueNameQueueRequest   = "bin-manager.queue-manager.request"
QueueNameQueueSubscribe = "bin-manager.queue-manager.subscribe"
```

### Registrar Manager
```go
QueueNameRegistrarEvent     = "bin-manager.registrar-manager.event"
QueueNameRegistrarRequest   = "bin-manager.registrar-manager.request"
QueueNameRegistrarSubscribe = "bin-manager.registrar-manager.subscribe"
```

### Route Manager
```go
QueueNameRouteEvent     = "bin-manager.route-manager.event"
QueueNameRouteRequest   = "bin-manager.route-manager.request"
QueueNameRouteSubscribe = "bin-manager.route-manager.subscribe"
```

### Schedule Manager
```go
QueueNameScheduleEvent     = "bin-manager.schedule-manager.event"
QueueNameScheduleRequest   = "bin-manager.schedule-manager.request"
QueueNameScheduleSubscribe = "bin-manager.schedule-manager.subscribe"
```

### Sentinel Manager
```go
QueueNameSentinelEvent     = "bin-manager.sentinel-manager.event"
QueueNameSentinelRequest   = "bin-manager.sentinel-manager.request"
QueueNameSentinelSubscribe = "bin-manager.sentinel-manager.subscribe"
```

### Storage Manager
```go
QueueNameStorageEvent     = "bin-manager.storage-manager.event"
QueueNameStorageRequest   = "bin-manager.storage-manager.request"
QueueNameStorageSubscribe = "bin-manager.storage-manager.subscribe"
```

### Tag Manager
```go
QueueNameTagEvent     = "bin-manager.tag-manager.event"
QueueNameTagRequest   = "bin-manager.tag-manager.request"
QueueNameTagSubscribe = "bin-manager.tag-manager.subscribe"
```

### Talk Manager
```go
QueueNameTalkEvent     = "bin-manager.talk-manager.event"
QueueNameTalkRequest   = "bin-manager.talk-manager.request"
QueueNameTalkSubscribe = "bin-manager.talk-manager.subscribe"
```

### Transcribe Manager
```go
QueueNameTranscribeEvent     = "bin-manager.transcribe-manager.event"
QueueNameTranscribeRequest   = "bin-manager.transcribe-manager.request"
QueueNameTranscribeSubscribe = "bin-manager.transcribe-manager.subscribe"
```

### Transfer Manager
```go
QueueNameTransferEvent     = "bin-manager.transfer-manager.event"
QueueNameTransferRequest   = "bin-manager.transfer-manager.request"
QueueNameTransferSubscribe = "bin-manager.transfer-manager.subscribe"
```

### TTS Manager
```go
QueueNameTTSEvent     = "bin-manager.tts-manager.event"
QueueNameTTSRequest   = "bin-manager.tts-manager.request"
QueueNameTTSSubscribe = "bin-manager.tts-manager.subscribe"
```

### User Manager
```go
QueueNameUserEvent     = "bin-manager.user-manager.event"
QueueNameUserRequest   = "bin-manager.user-manager.request"
QueueNameUserSubscribe = "bin-manager.user-manager.subscribe"
```

### Webhook Manager
```go
QueueNameWebhookEvent     = "bin-manager.webhook-manager.event"
QueueNameWebhookRequest   = "bin-manager.webhook-manager.request"
QueueNameWebhookSubscribe = "bin-manager.webhook-manager.subscribe"
```

## Exchanges

`QueueName` is a conflated type: some of its constants name queues, others name exchanges. The three exchanges below are shared infrastructure — no service owns them, and every one of them is declared by more than one process.

| Exchange | Kind | Audience | Routing key |
|----------|------|----------|-------------|
| `bin-manager.event` | topic | Internal services (cross-tenant) | `<publisher>.<resource>.<subscription-id>.<action>` |
| `bin-manager.webhook-manager.event.topic` | topic | Clients (customer/agent-scoped WebSocket + webhook delivery) | scope-first: `customer_id.<uuid>....` |
| `bin-manager.delay` | (delay plugin) | Internal services | delivery delay, not content routing |

Per-service exchanges (`bin-manager.<service>.event`) were fanout and the system of record during VOIP-1404-1406's dual-publish window. **VOIP-1407 completed the cutover: for every topic-enabled publisher, `bin-manager.event` is now the sole delivery path — no per-service fanout exchange is declared or published to by that instance.** The one surviving fanout-only publisher, `voip-asterisk-proxy`, has no topic counterpart and is untouched by this cutover; see the asterisk exception below.

### `bin-manager.event` — global topic exchange

Introduced by VOIP-1404 as an addition alongside the per-service fanout exchanges (dual publish); VOIP-1407 completed the cutover to topic-only. Carries internal service-to-service events. **For every topic-enabled publisher (`WithGlobalTopicPublish()` passed to `notifyhandler.NewNotifyHandler`), this is now the sole delivery path.** Before VOIP-1407, an event published through `notifyhandler` by a service that opted in was delivered here **in addition to** its per-service fanout exchange, with a byte-identical `sock.Event` payload on both — that dual delivery no longer happens; the per-service fanout leg is gone for these instances.

**Key schema**

```
<publisher>.<resource>.<subscription-id>.<action>

transcribe-manager.transcript.9f01c3d2-….created
transcribe-manager.transcribe.9f01c3d2-….speech_interim
call-manager.call.4a539340-….created
```

| Segment | Source | Rule |
|---------|--------|------|
| publisher | the publishing service name | normalized: lowercased, `.`/`*`/`#` → `_`, empty → `-` (no-op for all real service names) |
| resource | normalized `EventType` | first segment of `SplitN(type, "_", 2)`; `-` when the type has no `_` |
| subscription-id | the event data | the **subscription address** — see below; `-` when absent, `uuid.Nil`, or longer than 64 bytes (oversized addresses cannot be instance-subscribed) |
| action | normalized `EventType` | the remainder after the first `_` |

Normalization lowercases the event type and replaces `.` with `_` before splitting; any residual `.`/`*`/`#` in a computed segment becomes `_`, so a published key can never act as an accidental wildcard. Keys and binding patterns are generated by the same pure functions in `bin-common-handler/models/eventtopic` (`RoutingKey`, `PatternAll`, `PatternResource`, `PatternInstance`, `PatternAction`, `PatternForEventType`), so a pattern always matches the keys it was built for. `PatternForEventType(publisher, eventType)` (VOIP-1406 amendment) derives a `PatternAction` binding directly from the publisher's own canonical `EventType*` constant instead of a hand-typed resource/action literal -- the preferred call form for consumer `topicPatterns`, since a literal does not follow the constant if its value ever changes.

**Subscription address (third segment).** It answers "by which id will subscribers address this stream?", which is not always the resource's own id. Since VOIP-1419 the address comes from `eventtopic.SubscriptionIdentifier`, which is the parameter type of `notifyhandler.PublishEvent` (and, through `WebhookEventMessage`, of `PublishWebhookEvent`) -- a type that does not satisfy it does not compile at its publish site. `commonidentity.Identity` implements the interface, so every model embedding Identity by value gets the own-id default through method promotion; an explicit **pointer-receiver** method is written only where the address is not the own id (parent-stream overrides, nil-guarded wrappers, types without an Identity embed). There is no JSON fallback: the marshaled payload's top-level `id` plays no role in resolution, and the only degrade path is an explicit empty (or uuid.Nil) return → the `-` placeholder. Most resources return their own id; stream-child resources return their parent's (example: every transcribe-manager event -- `transcribe`, `streaming`, and `transcript` alike -- carries the transcribe-id, so one session is followed across three namespaces). A non-Nil-but-meaningless id is worse than no id: it produces well-formed keys that match nothing and evades the placeholder metric, so each publisher pins its keys with a golden-key test (`bin-transcribe-manager/models/transcribe/routingkey_golden_test.go` is the template).

**Per-publisher parent-stream addresses (VOIP-1405).** Every type not listed here returns its OWN id through the `EventSubscriptionID()` promoted from the embedded `commonidentity.Identity` (or, for the handful of published types without the embed, an explicit own-id method). VOIP-1419 made the contract mandatory for all published types; this table lists only the ones whose address is NOT their own id. Two categories (design: `docs/plans/2026-08-27-voip-1405-topic-publisher-rollout-design.md` §2):

| Publisher | Type(s) | Address | Category |
|---|---|---|---|
| ai-manager | `message.Message`, `message.IntermediateWebhookMessage` | AIcallID | B / A |
| call-manager | `dtmf.DTMF`, `call.OutboundWhitelistRejectedEvent` | CallID | A |
| campaign-manager | `campaigncall.Campaigncall` | CampaignID | B |
| conference-manager | `conferencecall.Conferencecall` | ConferenceID | B |
| contact-manager | `casenote.CaseNote`, `casenote.CaseNoteDeletedEvent`, `kase.CaseTagEvent`, `kase.CaseContactEvent` | CaseID | B / B / A / A |
| conversation-manager | `message.Message` | ConversationID | B |
| pipecat-manager | `message.Message`, `message.MemberSwitchedEvent` | PipecatcallID | A |
| queue-manager | `queuecall.Queuecall` | QueueID | B |
| schedule-manager | `execution.Execution` | ScheduleID | B |
| talk-manager | `message.Message`, `participant.Participant` | ChatID | B |
| transcribe-manager | `streaming.Speech`, `streaming.Streaming`, `transcript.Transcript` | TranscribeID | A/A/B (pilot) |
| tts-manager | `message.Message` | StreamingID | A |
| webchat-manager | `message.Message` | SessionID | B |

Category A = own id structurally unusable (per-event/stale/absent). Category B = own id stable but the parent stream is the natural consumption axis; adopting the parent address deliberately forfeits own-id instance subscription for that child (the child id first appears in its `created` event, so own-id pre-binding has no value; single-item retrieval stays available via RPC).

**Address-convergence (what one pattern set follows):** tts `streaming`+`message` → one streaming-id; contact `case` namespace (`contact-manager.case.<case-id>.#`) → whole case lifecycle; pipecat `pipecatcall`+`message`+`team` → one pipecatcall-id; webchat — both event types collapse to resource `webchat` with the session address, so `webchat-manager.webchat.<session-id>.#` follows a whole session; talk `chat`+`chatmessage`+`chatparticipant` → one chat-id; conversation lifecycle + messages → one conversation-id.

**Deliberate own-id / placeholder addresses (VOIP-1419 revision of the former "do not fix" list):** every hazard in the old list is now resolved deliberately, not ignored -- by an explicit method where promotion cannot apply (`customer.Customer`, `CustomerCreatedEvent`, `accesskey.Accesskey`, sentinel `pod.Event`), and by the own-id default promoted from the embedded `commonidentity.Identity` for the rest (`activeflow.Activeflow`, `billing.Billing`, `recording.Recording`). `customer.Customer` returns its own `ID` field, and `CustomerCreatedEvent` carries its OWN explicit method with a nil-embed guard that SHADOWS any promotion (the promotion trap the old note warned about cannot fire past an explicit method at depth 0; the customer golden pins both branches). `activeflow.Activeflow` deliberately resolves to its own id via the promoted default, never `ReferenceID` (which can be Nil) -- pinned by both a golden row and a behavioral test. `billing.Billing` and `recording.Recording` resolve to their own pre-bindable ids the same promoted way; `accesskey.Accesskey` returns its own id from its explicit method (no Identity embed). Sentinel is **no longer** a placeholder-by-design publisher as of VOIP-1418: `models/pod` (and its `*corev1.Pod` payload, which carried no top-level `id`) is deleted, and `container.Event.EventSubscriptionID()` returns the resolved asterisk-id, so `container_died` events publish a real, bindable address. The `-` placeholder now appears only for every `container_started` (a freshly started container has no resolved id yet) and for a `container_died` whose id was never resolvable; `sentinel_manager_topic_placeholder_total ≈ topic_publish_total{ok}` is therefore NO LONGER the healthy invariant for this service -- a rising placeholder rate on `died` keys specifically is a degraded state, cross-checked against `sentinel_manager_container_unresolved_asterisk_id_total`. Instance subscription of container events is now possible; nothing binds that way today. The routing keys changed with it, deliberately and reviewed: `sentinel-manager.pod.-.updated`/`.deleted` -> `sentinel-manager.container.<asterisk-id>.started`/`.died` (pinned by `bin-sentinel-manager/models/container/routingkey_golden_test.go`, and on the consumer side by `bin-call-manager/pkg/subscribehandler/binding_golden_test.go`).

**Declaration invariant.**

- Declare **only** through the shared helper: `sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic")`. `durable=true` / `autoDelete=false` are hardcoded in `rabbitmqhandler/topic.go`. Hand-rolled declarations are forbidden — an AMQP redeclare is idempotent only when every parameter matches, and a mismatch closes the channel with `406 PRECONDITION_FAILED`.
- **Both sides declare, idempotently**: publishers declare at construction time (via `notifyhandler.WithGlobalTopicPublish()`), and subscribers declare at startup too. Subscriber-side declaration is what makes start order irrelevant — a consumer that boots before any publisher must not fail its `QueueBind`. Since VOIP-1407 a subscriber-side declare/bind failure is fatal (see "Consumer state" below) — there is no fanout `QueueSubscribe` fallback left for a consumer to fall back on either.
- **A declare failure is now FATAL (VOIP-1407), not a degrade.** Through VOIP-1406, a publisher-side topic-exchange declare failure degraded the handler: it stayed alive, the fanout publish kept working untouched, and every suppressed topic publish incremented `<ns>_topic_publish_total{result="error"}`. VOIP-1407 removed the fanout fallback that made degrading safe. Now a declare failure on either side — the fanout exchange inside `NewNotifyHandler` (for the sole remaining fanout-only construction, `voip-asterisk-proxy`), or the global topic exchange inside `initGlobalTopicExchange` (for every topic-enabled instance) — calls `logrus.Fatalf` and halts the process at startup instead. There is no `topicDisabled` field, no suppressed-publish counting, and no silent degrade path left anywhere in `notifyhandler`.

**Consumer state (VOIP-1406/VOIP-1407).** All 20 consumer services (the 21 subscribers minus api-manager, which consumes the client-facing VOIP-1258 exchange) bind `bin-manager.event` with `eventtopic.PatternForEventType` patterns matching exactly their dispatch sets (timeline-manager binds `#`; the other 19 derive every pattern from the publisher's own `EventType*` constant rather than a hand-typed literal -- VOIP-1406 amendment). **`topicPatterns`/`QueueBind` is now the SOLE intake mechanism for inter-service events in all 20 services (VOIP-1407).** The fanout `QueueSubscribe` loop, the `subscribeTargets`/`fanoutUnbindTargets` lists that fed it and its unbind step, and (for call-manager/timeline-manager) the sentinel defensive `TopicCreate` declare have all been deleted -- there is no more "stay on fanout" rollback/degrade surface. A topic-exchange declare or topic-pattern bind failure at consumer startup is now FATAL: `Run()` returns the error immediately (or `nil, err` for timeline-manager's two-value signature), and the orchestrator's normal pod-restart-with-backoff handles recovery -- the same fatal-on-failure rule as the publish side above. Each service's exact pattern set is pinned by a `binding_golden_test.go` in its `pkg/subscribehandler`.

Two narrow exceptions survive the VOIP-1407 cutover untouched, in two different (overlapping) service pairs:

**Asterisk fanout exception (permanent, not migration debris).** `asterisk.all.event` and its publisher, `voip-asterisk-proxy`, are explicitly and permanently EXCLUDED from the VOIP-1407 topic cutover by deliberate design decision -- this is not leftover fanout debris awaiting a future cleanup ticket. `voip-asterisk-proxy`'s publish side (`voip-asterisk-proxy/cmd/asterisk-proxy/main.go`) has no topic counterpart and is completely untouched by VOIP-1407: it constructs its `NotifyHandler` without `WithGlobalTopicPublish()` and stays on the pre-existing fanout-only path -- the ONLY `topicEnabled=false` `NewNotifyHandler` construction left anywhere in the monorepo after VOIP-1407. Its two consumers, call-manager and timeline-manager, retain a standalone fanout subscription in `Run()`:

```go
// The asterisk fanout leg is permanently retained: asterisk-proxy does not publish to the
// global topic exchange, so this is the one fanout QueueSubscribe that survives the cutover.
if errSubscribe := h.sockHandler.QueueSubscribe(string(h.subscribeQueue), string(commonoutline.QueueNameAsteriskEventAll)); errSubscribe != nil {
    return fmt.Errorf("could not subscribe to the asterisk fanout exchange. err: %v", errSubscribe)
}
```

positioned immediately after `QueueCreate`, before the topic-declare line -- independent of, and not part of, the (now-deleted) generic fanout-target loop. `asterisk.all.event` is not part of the 28-exchange deletion set in the runbook below, and there is no plan to migrate it to the topic exchange.

**VOIP-1258 webhook-topic-bind exception (agent-manager, timeline-manager only).** These two services -- not call-manager, not any of the other 17 -- additionally carry a self-contained block in `Run()`, predating and unrelated to the VOIP-1406/1407 fanout-vs-topic cutover:

```go
if errBind := h.sockHandler.QueueBind(h.subscribeQueue, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); errBind != nil {
    log.Errorf("Could not bind to the topic exchange. err: %v", errBind)
    // do NOT proceed to unbind the old exchange if this bind failed -- stay on the
    // old exchange rather than risk ending up bound to neither.
} else if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", string(commonoutline.QueueNameWebhookEvent), nil); errUnbind != nil {
    log.Errorf("CRITICAL: Could not unbind from the old fanout exchange after binding to the new topic exchange. ... err: %v", errUnbind)
}
```

This block binds to the VOIP-1258 client-facing topic exchange (`bin-manager.webhook-manager.event.topic` -- a DIFFERENT exchange from the VOIP-1406/1407 internal topic exchange `bin-manager.event`, see below) and unbinds the legacy `bin-manager.webhook-manager.event` fanout exchange, which is already absent from the broker (see the exclusion list in the runbook below). **Its failure semantics are deliberately still log-only and non-fatal -- this is the one deliberate exception to "everything is fatal now" in this document.** VOIP-1407 promoted the VOIP-1406 topic-pattern block's failure semantics to fatal specifically because that block lost its fanout fallback; this VOIP-1258 block's own fallback (staying bound to the legacy `QueueNameWebhookEvent` exchange on a failed bind) is untouched by VOIP-1407 and remains a functioning degrade path, so its failure handling was deliberately left exactly as it was -- not promoted to `logrus.Fatalf`. The legacy `QueueUnbind(..., QueueNameWebhookEvent, ...)` call is likewise left untouched: cleaning it up is a separate, narrower VOIP-1258 concern, out of scope for VOIP-1407.

**Stale-binding runbook (rollback and rolling-deploy windows).** Two triggers, one policy: (a) rolling back a migrated service re-subscribes fanout while the topic bindings persist on the durable queue; (b) during a rolling deploy of a 2-replica service, an old-image pod hitting a broker reconnect after the new pod unbound fanout replays its tracked fanout bind. Both end states are double delivery on the shared queue, which is TOLERATED (at-least-once was always the contract). Detect and remediate from bm-nyc-01:

```
# inspect a queue's bindings (expect: topic patterns on bin-manager.event; no empty-key fanout bind post-migration)
pass=$(docker exec infra-rabbitmq printenv RABBITMQ_DEFAULT_PASS); ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' infra-rabbitmq | awk '{print $1}')
curl -s -u voipbin:$pass "http://$ip:80/api/queues/%2F/bin-manager.<svc>.subscribe/bindings" | jq -r '.[] | "\(.source) \(.routing_key)"'

# remove a stale fanout binding manually (empty routing key)
curl -s -u voipbin:$pass -X DELETE "http://$ip:80/api/bindings/%2F/e/bin-manager.<publisher>.event/q/bin-manager.<svc>.subscribe/~"
```

Alternatively roll forward (redeploy the migrated image); its Run() re-unbinds fanout on the next boot.

**VOIP-1407 note: both a PRE-rollout AND a POST-rollout sweep are now mandatory, not just pre-rollout.** Before VOIP-1407's consumer-side change deploys to a given service, confirm zero stray fanout bindings for that service using the inspection command above (same as always). **After it deploys, re-confirm.** This second sweep matters specifically because of what VOIP-1407 deletes: through VOIP-1406, every service's `Run()` re-ran the bind-then-unbind sequence on every boot, so a stray fanout binding left by a rollback or a rolling-deploy race got automatically swept the next time that service restarted for ANY reason -- no human required. VOIP-1407 deletes that `QueueUnbind` loop entirely, along with the rest of the fanout-unbind machinery. Once it's gone, a stray binding present at the moment this service's VOIP-1407 code deploys has NO automatic remediation path left -- only a human running the manual `curl` procedure above. A pre-rollout sweep alone cannot catch a stray created DURING the rollout window itself (an old-image pod's `Run()` re-subscribing to fanout mid-rollout); only the post-rollout sweep catches that.

**Exchange deletion runbook (VOIP-1407, one-time, post-merge, post-deployment operational step -- NOT part of this PR's code).** Once the fanout-cutover code above (both the publish-side `bin-common-handler/pkg/notifyhandler` change and the consumer-side `pkg/subscribehandler` change in all 20 services) is fully redeployed, the 28 now-orphaned per-service fanout exchanges below become safe to delete from the broker. This is a manual, one-time administrative cleanup -- it does not run as part of CI/CD and is not automated.

*Precondition (MUST be satisfied before running any command below):*

- Every publisher daemon and its `-control` CLI binary (the ~56 topic-only publish call sites) has been rebuilt and redeployed with the fanout-publish code removed.
- All 20 VOIP-1406 consumer services have been rebuilt and redeployed with the fanout `QueueSubscribe` loop, `subscribeTargets`, and `fanoutUnbindTargets`/`QueueUnbind` removed.
- For every service above, a CONFIRMED restart-survival check has been performed -- not merely a point-in-time binding snapshot. Restart each service (or wait for its next natural restart) and re-inspect its queue bindings to confirm no remaining code path re-declares or re-subscribes to any of the 28 exchanges below.
- The pre-rollout AND post-rollout stale-binding sweeps above have found zero stray fanout bindings for every affected service. This precondition is not optional: once this service's fanout-unbind machinery is gone, a stray binding has no automatic remediation path left besides the manual `curl` procedure documented above.

*Mechanism.* A documented one-time cleanup via the RabbitMQ management HTTP API, **on port 80** (NOT 15672, and NOT `rabbitmqadmin`) on bm-nyc-01 -- the same tool and pattern as the "Stale-binding runbook" above. Per-exchange command form:

```bash
curl -u "$RABBITMQ_USER:$RABBITMQ_PASS" -X DELETE "http://<host>/api/exchanges/%2f/bin-manager.<x>-manager.event"
```

*All 28 exchanges to delete, enumerated verbatim (do not re-derive or abbreviate by range):*

```
bin-manager.ai-manager.event
bin-manager.agent-manager.event
bin-manager.billing-manager.event
bin-manager.call-manager.event
bin-manager.campaign-manager.event
bin-manager.conference-manager.event
bin-manager.contact-manager.event
bin-manager.conversation-manager.event
bin-manager.customer-manager.event
bin-manager.direct-manager.event
bin-manager.email-manager.event
bin-manager.flow-manager.event
bin-manager.message-manager.event
bin-manager.number-manager.event
bin-manager.outdial-manager.event
bin-manager.pipecat-manager.event
bin-manager.queue-manager.event
bin-manager.registrar-manager.event
bin-manager.route-manager.event
bin-manager.schedule-manager.event
bin-manager.sentinel-manager.event
bin-manager.storage-manager.event
bin-manager.tag-manager.event
bin-manager.talk-manager.event
bin-manager.transcribe-manager.event
bin-manager.transfer-manager.event
bin-manager.tts-manager.event
bin-manager.webchat-manager.event
```

Equivalently, as the `for` loop used to verify these 28 exist and have zero queue bindings (`tasks/todo.md` §1, the normative enumeration this list is copied from verbatim):

```bash
for ex in ai-manager agent-manager billing-manager call-manager campaign-manager \
    conference-manager contact-manager conversation-manager customer-manager \
    direct-manager email-manager flow-manager message-manager number-manager \
    outdial-manager pipecat-manager queue-manager registrar-manager route-manager \
    schedule-manager sentinel-manager storage-manager tag-manager talk-manager \
    transcribe-manager transfer-manager tts-manager webchat-manager; do
  curl -u "$RABBITMQ_USER:$RABBITMQ_PASS" -X DELETE \
    "http://<host>/api/exchanges/%2f/bin-manager.$ex.event"
done
```

*Explicitly NOT deleted by this runbook* -- five categories, none of them part of the 28 above:

- `asterisk.all.event` -- permanent, the excluded asterisk fanout leg (see above).
- `bin-manager.event` -- the global topic exchange itself, the migration target, not a fanout leftover.
- `bin-manager.webhook-manager.event.topic` -- a DIFFERENT exchange (VOIP-1258's client-facing topic exchange), not part of the fanout family at all.
- `bin-manager.delay` -- the unrelated delayed-message retry exchange.
- Five `QueueName*Event`-family exchanges already absent from the broker, so there is nothing to delete either way: `bin-manager.api-manager.event`, `bin-manager.rag-manager.event`, `bin-manager.timeline-manager.event`, and `bin-manager.user-manager.event` (all four: dead constants, never had an exchange -- api-manager/rag-manager/timeline-manager/user-manager never published fanout events), plus `bin-manager.webhook-manager.event` (absent from the broker; cause not established -- VOIP-1296 removed webhook-manager's own declare of it, but that alone does not explain a durable exchange's disappearance; not relied on by this runbook either way, since there is nothing to delete regardless of cause).

*Post-deletion re-check.* Re-list exchanges and bindings after a defined soak window (same technique as the precondition check above) to catch resurrection -- e.g. from an un-redeployed `-control` CLI binary invoked between the deploy and this deletion running. `-control` binaries run ad hoc rather than as long-running pods, so a stale binary invoked any time after deploy but before deletion is a live resurrection risk independent of the pod-restart risk the precondition already covers.

**Binding.** Consumers combine the pattern builders with the existing `sockhandler.QueueBind` / `QueueUnbind`. A broker binding is shared by every logical subscriber on the same queue+pattern: one `QueueUnbind` severs all of them. Anything multiplexing several logical subscribers over one queue must keep refcount discipline and bind/unbind only on the 0↔1 transition, the way `bin-api-manager/pkg/websockhandler/scoperefcount.go` does.

**Consumer contract — the bind-after-start race.** A consumer that learns the subscription address from an RPC response can miss the first events, because the publisher may already be streaming by the time the response returns. Concretely: `transcribe start` generates the transcribe-id inside `startLive` and begins streaming before returning, and unroutable topic messages are dropped silently (`mandatory=false`). Resolution is decided per consumer at adoption time:

- **(a) State repair after bind — recommended default.** After binding, call the existing list API once (`TranscribeV1TranscriptList` for transcribe-manager) to backfill whatever was missed. No API change, works today.
- **(b) Client-generated id.** Let the caller supply the resource id so it can bind *before* starting the resource — lossless, but requires an API change. Future option, not available today.
- **(c) Documented initial-loss tolerance.** Acceptable where the missed events are ephemeral by nature, e.g. interim speech results that are superseded within seconds.

The routing-key schema is independent of this choice; nothing in the exchange needs to change when a consumer switches strategies.

### `bin-manager.webhook-manager.event.topic` — client-facing topic exchange

Introduced by VOIP-1258. Different audience, different key shape, and it is **not** being replaced by `bin-manager.event` — the two coexist permanently.

- Keys are **scope-first** (`customer_id.<uuid>….`) because the consumer is a client connection that may only ever see one tenant's traffic. Scope is the first thing a binding must constrain.
- `bin-manager.event` keys carry **no tenant segment by design**: internal services operate cross-tenant, and a customer segment there would only add a wildcard everyone has to write.
- Published through `notifyhandler.PublishEventWithRoutingKey`, on a handler built with `NewNotifyHandlerForExistingExchange` (the caller declares the exchange, because the constructor's own declare is fanout-kind). `WithGlobalTopicPublish()` must **not** be enabled on webhook-manager's scope-first instance — it would triple-publish webhook events.

### `bin-manager.delay` — delayed delivery

Not a routing exchange in the content sense: it exists so a message can be published now and delivered later (`publishDelayedEvent`). It is the precedent for a two-segment, service-agnostic global exchange name, which is why the new global exchange is `bin-manager.event` rather than a three-segment per-service name. Delayed publishes do **not** dual-publish to `bin-manager.event`. VOIP-1405 closed the deferred question as **not applicable**: no public API produces `delay>0` today, so the `delay == 0` guard in the topic publish path is purely defensive.

## Usage Examples

### ListenHandler (Receiving Requests)

```go
// In pkg/listenhandler/main.go
func (h *listenHandler) Run(queue, exchangeDelay string) error {
    // queue = "bin-manager.call-manager.request"
    if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
        return fmt.Errorf("could not declare the queue: %v", err)
    }

    go h.sockHandler.ConsumeRPC(ctx, queue, serviceName, false, false, false, 10, h.processRequest)
    return nil
}
```

### NotifyHandler (Publishing Events)

```go
// Publishing an event
h.notifyHandler.PublishEvent(ctx, outline.QueueNameCallEvent, call.EventCallCreated, callData)
```

### SubscribeHandler (Subscribing to Events)

```go
// In pkg/subscribehandler/main.go
func (h *subscribeHandler) Run() error {
    // Subscribe to customer events
    h.sockHandler.QueueBind(string(outline.QueueNameCallSubscribe), string(outline.QueueNameCustomerEvent))

    // Subscribe to flow events
    h.sockHandler.QueueBind(string(outline.QueueNameCallSubscribe), string(outline.QueueNameFlowEvent))

    go h.sockHandler.ConsumeEvent(ctx, string(outline.QueueNameCallSubscribe), serviceName, h.processEvent)
    return nil
}
```

### RequestHandler (Making RPC Calls)

```go
// Making a request to another service
resp, err := h.reqHandler.CallV1CallGet(ctx, callID)
// Internally sends to: bin-manager.call-manager.request
```

## Queue Types

### Normal Queues
- Durable, survive broker restarts
- Used for request/response and persistent events
- Created with `QueueCreate(queue, "normal")`

### Volatile Queues
- Auto-delete when unused
- Used for temporary subscriptions
- Created with `QueueCreate(queue, "volatile")`

## Message Format

### Request (sock.Request)
```go
type Request struct {
    URI       string      `json:"uri"`        // e.g., "/v1/calls/{uuid}"
    Method    string      `json:"method"`     // GET, POST, PUT, DELETE
    Publisher string      `json:"publisher"`  // Source service name
    Data      interface{} `json:"data"`       // Request payload
}
```

### Response (sock.Response)
```go
type Response struct {
    StatusCode int         `json:"status_code"` // HTTP-style status code
    DataType   string      `json:"data_type"`   // Response type identifier
    Data       interface{} `json:"data"`        // Response payload
}
```

### Event (sock.Event)
```go
type Event struct {
    Type      string      `json:"type"`      // Event type (e.g., "call.created")
    Publisher string      `json:"publisher"` // Source service name
    Data      interface{} `json:"data"`      // Event payload
}
```

## See Also

- [Architecture Deep Dive](architecture-deep-dive.md) - Service communication patterns
- [Common Workflows](common-workflows.md) - Adding new endpoints
- `bin-common-handler/pkg/requesthandler/` - RPC request implementations
- `bin-common-handler/pkg/notifyhandler/` - Event publishing implementations
