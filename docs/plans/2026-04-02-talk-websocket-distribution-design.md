# Design: Fix Talk WebSocket Distribution Topic

**Date:** 2026-04-02
**Branch:** NOJIRA-Fix-aimessage-websocket-distribution

## Problem Statement

Talk events (messages, chats, participants) are handled by a separate code path (`talk.go`) from all other webhook events (`webhookmanager.go`). This is unnecessary duplication since talk events already flow through webhook-manager as `webhook_published` events.

Additionally, the old-format ZMQ topics for talk events use the resource's own ID:

```
agent_id:{ownerID}:{resource}:{resource_id}
```

Clients need to subscribe by chat ID to receive all events for a given chat.

## Goal

1. Merge talk event processing into the webhook path (`webhookmanager.go`)
2. Change the old-format topic for talk events to per-chat:

```
agent_id:{ownerID}:chat:{chat_id}
customer_id:{customerID}:chat:{chat_id}
```

3. Remove `talk.go` and the direct talk-manager routing in `main.go`

## Current Flow

```
Talk events arrive via TWO paths:

Path 1 (direct): talk-manager → api-manager processEventTalkManager
  → Parses specific talk structs, fans out to all participants
  → Publishes agent_id topics to ZMQ

Path 2 (webhook): talk-manager → webhook-manager → api-manager processEventWebhookManagerWebhookPublished
  → Parses commonWebhookData, generates generic topics
  → Publishes customer_id topics to ZMQ
```

## Change

Merge both paths into the webhook path. In `webhookmanager.go`:

### 1. Add `ChatID` to `commonWebhookData`

```go
type commonWebhookData struct {
    commonidentity.Identity
    commonidentity.Owner
    AIcallID uuid.UUID `json:"aicall_id,omitempty"`
    ChatID   uuid.UUID `json:"chat_id,omitempty"`
}
```

### 2. Add `ctx` parameter to `createTopics`

Required for the participant fan-out RPC call.

```go
func (h *subscribeHandler) createTopics(ctx context.Context, messageType string, d *commonWebhookData, publisher string) ([]string, error)
```

Update the caller in `processEventWebhookManagerWebhookPublished`:

```go
topics, err := h.createTopics(ctx, whData.Type, d, m.Publisher)
```

### 3. Modify old-format topic generation in `createTopics` to switch on resource type

```go
resource := tmps[0]

switch resource {
case "aimessage":
    if d.CustomerID != uuid.Nil {
        res = append(res, fmt.Sprintf("customer_id:%s:aicall:%s", d.CustomerID, d.AIcallID))
    }

case "chat", "chatmessage", "chatparticipant":
    // For chat events, the resource IS the chat (no separate chat_id field)
    chatID := d.ChatID
    if chatID == uuid.Nil {
        chatID = d.ID
    }

    if d.CustomerID != uuid.Nil {
        res = append(res, fmt.Sprintf("customer_id:%s:chat:%s", d.CustomerID, chatID))
    }
    if d.OwnerID != uuid.Nil {
        res = append(res, fmt.Sprintf("agent_id:%s:chat:%s", d.OwnerID, chatID))
    }

    // Fan-out to all chat participants
    if chatID != uuid.Nil {
        participants, err := h.reqHandler.TalkV1ParticipantList(ctx, chatID)
        if err == nil {
            for _, p := range participants {
                if p.OwnerID == d.OwnerID {
                    continue
                }
                // Old format
                res = append(res, fmt.Sprintf("agent_id:%s:chat:%s", p.OwnerID, chatID))
                // New format
                res = append(res, fmt.Sprintf("agent_id:%s:%s:%s:%s", p.OwnerID, service, messageType, d.ID))
            }
        }
    }

default:
    if d.CustomerID != uuid.Nil {
        res = append(res, fmt.Sprintf("customer_id:%s:%s:%s", d.CustomerID, resource, d.ID))
    }
    if d.OwnerID != uuid.Nil {
        res = append(res, fmt.Sprintf("agent_id:%s:%s:%s", d.OwnerID, resource, d.ID))
    }
}
```

New-format topics remain unchanged (generated after the switch).

### 4. Remove talk-manager routing in `main.go`

Remove:
```go
case m.Publisher == string(commonoutline.ServiceNameTalkManager):
    err = h.processEventTalkManager(ctx, m)
```

### 5. Delete `talk.go`

Remove the entire file: `processEventTalkManager`, `processEventTalkMessage`, `processEventTalk`, `processEventTalkParticipant`, `createTalkTopics`, `getTalkParticipants`, `extractResource`.

### 6. Update tests

- Move relevant test cases from `talk_test.go` to `webhookmanager_test.go`
- Add test cases for chat, chatmessage, chatparticipant events with participant fan-out
- Add test case for chat events where `ChatID` is nil (falls back to `d.ID`)
- Remove `talk_test.go`

## Why This Is Safe

- **Talk events already flow through webhook-manager**: Removing the direct path removes duplication, not functionality.
- **Non-talk events unaffected**: The `switch` on resource type means only `chat`, `chatmessage`, `chatparticipant` prefixes trigger the new logic. All other events follow the default path.
- **WebSocket validation unaffected**: `validateTopic` only checks the `customer_id:<uuid>` or `agent_id:<uuid>` prefix.
- **ChatID availability**: Message and participant webhook data include `chat_id`. Chat webhook data uses its own `id` as the chat ID (handled by the `chatID == uuid.Nil` fallback).

## Files to Change

| File | Change |
|------|--------|
| `bin-api-manager/pkg/subscribehandler/webhookmanager.go` | Add `ChatID` field, add `ctx` param, add resource-type switch with participant fan-out |
| `bin-api-manager/pkg/subscribehandler/webhookmanager_test.go` | Add talk event test cases with participant fan-out |
| `bin-api-manager/pkg/subscribehandler/main.go` | Remove talk-manager routing case |
| `bin-api-manager/pkg/subscribehandler/talk.go` | Delete |
| `bin-api-manager/pkg/subscribehandler/talk_test.go` | Delete |
