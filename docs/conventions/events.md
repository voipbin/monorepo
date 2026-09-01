# Event Publishing

### 11.1 PublishWebhookEvent

Use `notifyHandler.PublishWebhookEvent()` for both internal event and customer webhook:

```go
// CORRECT — fires both internal event + customer webhook
h.notifyHandler.PublishWebhookEvent(ctx, agent.CustomerID, agent.EventTypeAgentCreated, agent)

// CORRECT — internal event only (no customer webhook)
h.notifyHandler.PublishEvent(ctx, agent.EventTypeAgentStatusUpdated, agent)
```

### 11.2 Fire-and-Forget

Events are published asynchronously via goroutines. Do not wait for event delivery:

```go
// This is how PublishWebhookEvent works internally:
go h.PublishEvent(ctx, eventType, data)      // async
go h.PublishWebhook(ctx, customerID, eventType, data)  // async
```

### 11.3 Event Handler Return Values

Event handlers must return `nil` to acknowledge the message. Returning an error requeues the message:

```go
// CORRECT — return nil to acknowledge, even on handled errors
func (h *handler) EventCMCallHangup(ctx context.Context, call *cmcall.Call) error {
    if err := h.processHangup(ctx, call); err != nil {
        log.Errorf("Could not process hangup: %v", err)
        return nil  // Acknowledge — don't requeue on business logic errors
    }
    return nil
}
```

---
