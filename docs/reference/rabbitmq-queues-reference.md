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

Per-service exchanges (`bin-manager.<service>.event`) are fanout and remain the system of record while dual publish lasts.

### `bin-manager.event` — global topic exchange

Introduced by VOIP-1404. Carries internal service-to-service events. Every event published through `notifyhandler` by a service that opted in is delivered here **in addition to** its per-service fanout exchange; the `sock.Event` payload is byte-identical on both.

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

Normalization lowercases the event type and replaces `.` with `_` before splitting; any residual `.`/`*`/`#` in a computed segment becomes `_`, so a published key can never act as an accidental wildcard. Keys and binding patterns are generated by the same pure functions in `bin-common-handler/models/eventtopic` (`RoutingKey`, `PatternAll`, `PatternResource`, `PatternInstance`, `PatternAction`), so a pattern always matches the keys it was built for.

**Subscription address (third segment).** It answers "by which id will subscribers address this stream?", which is not always the resource's own id. The default is the top-level `id` of the event payload. A resource whose own id is regenerated per event (or addresses the wrong thing) overrides it by implementing `eventtopic.SubscriptionIdentifier` with a **pointer receiver**. Example: every transcribe-manager event — `transcribe`, `streaming`, and `transcript` alike — carries the transcribe-id, so one session is followed across three namespaces. A non-Nil-but-meaningless id is worse than no id: it produces well-formed keys that match nothing and evades the placeholder metric, so each publisher pins its keys with a golden-key test (`bin-transcribe-manager/models/transcribe/routingkey_golden_test.go` is the template).

**Declaration invariant.**

- Declare **only** through the shared helper: `sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic")`. `durable=true` / `autoDelete=false` are hardcoded in `rabbitmqhandler/topic.go`. Hand-rolled declarations are forbidden — an AMQP redeclare is idempotent only when every parameter matches, and a mismatch closes the channel with `406 PRECONDITION_FAILED`.
- **Both sides declare, idempotently**: publishers declare at construction time (via `notifyhandler.WithGlobalTopicPublish()`), and subscribers declare at startup too. Subscriber-side declaration is what makes start order irrelevant — a consumer that boots before any publisher must not fail its `QueueBind`.
- A declare failure on the publisher side degrades rather than aborts: the handler stays alive, the fanout publish is untouched, and every suppressed topic publish increments `<ns>_topic_publish_total{result="error"}`.

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

Not a routing exchange in the content sense: it exists so a message can be published now and delivered later (`publishDelayedEvent`). It is the precedent for a two-segment, service-agnostic global exchange name, which is why the new global exchange is `bin-manager.event` rather than a three-segment per-service name. Delayed publishes do **not** dual-publish to `bin-manager.event`; delayed-event topic semantics are deferred to the follow-up that opts the remaining publishers in.

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
