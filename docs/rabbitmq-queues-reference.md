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

### Chat Manager
```go
QueueNameChatEvent     = "bin-manager.chat-manager.event"
QueueNameChatRequest   = "bin-manager.chat-manager.request"
QueueNameChatSubscribe = "bin-manager.chat-manager.subscribe"
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
