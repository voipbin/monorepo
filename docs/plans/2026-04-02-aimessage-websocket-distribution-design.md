# Design: Fix aimessage WebSocket Distribution Topic

**Date:** 2026-04-02
**Branch:** NOJIRA-Fix-aimessage-websocket-distribution

## Problem Statement

The current ZMQ topic for aimessage webhook events uses the message's own ID:

```
customer_id:<customer-id>:aimessage:<aimessage-id>
```

Clients cannot subscribe to this topic because they don't know a message's ID before it exists. They need to subscribe by AI call ID to receive all messages (`aimessage_created` and `aimessage_intermediate`) for a given AI call.

## Goal

Change the old-format ZMQ topic for aimessage events from per-message to per-aicall:

```
customer_id:<customer-id>:aicall:<aicall-id>
```

The new-format topic (service-namespaced) is being removed soon and is not changed.

## Current Flow

```
ai-manager publishes aimessage webhook
  → webhook-manager receives, publishes webhook_published event
  → api-manager processEventWebhookManagerWebhookPublished
  → createTopics extracts resource from event type, ID from commonWebhookData
  → Old format: customer_id:<cid>:aimessage:<msg-id>
  → Publishes to ZMQ
  → WebSocket clients receive if subscribed to matching topic
```

## Change

In `bin-api-manager/pkg/subscribehandler/webhookmanager.go`:

### 1. Add `AIcallID` to `commonWebhookData`

```go
type commonWebhookData struct {
    commonidentity.Identity
    commonidentity.Owner
    AIcallID uuid.UUID `json:"aicall_id,omitempty"`
}
```

The `aicall_id` field already exists in both `WebhookMessage` and `IntermediateWebhookMessage` JSON payloads from ai-manager. This just enables parsing it.

### 2. Modify old-format topic in `createTopics`

When `d.AIcallID` is set (non-nil), use `aicall:<aicall-id>` instead of `<resource>:<id>`:

```go
if d.CustomerID != uuid.Nil {
    if d.AIcallID != uuid.Nil {
        res = append(res, fmt.Sprintf("customer_id:%s:aicall:%s", d.CustomerID, d.AIcallID))
    } else {
        res = append(res, fmt.Sprintf("customer_id:%s:%s:%s", d.CustomerID, resource, d.ID))
    }
}
```

### 3. Update tests

Add test cases for aimessage events with `aicall_id` in webhook data.

## Why This Is Safe

- **Non-aimessage events unaffected**: Only ai-manager message webhooks include `aicall_id`. All other events parse it as `uuid.Nil` and follow existing path.
- **WebSocket validation unaffected**: `validateTopic` only checks the `customer_id:<uuid>` prefix.
- **No ai-manager or pipecat-manager changes needed**: The `aicall_id` is already in the webhook JSON payloads.

## Files to Change

| File | Change |
|------|--------|
| `bin-api-manager/pkg/subscribehandler/webhookmanager.go` | Add `AIcallID` field, modify old-format topic logic |
| `bin-api-manager/pkg/subscribehandler/webhookmanager_test.go` | Add aimessage test cases |
