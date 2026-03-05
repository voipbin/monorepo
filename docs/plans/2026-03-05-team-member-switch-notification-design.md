# Team Member Switch Notification Design

**Date:** 2026-03-05
**Branch:** NOJIRA-Add-team-member-switch-notification

## Problem Statement

When an AI call uses a team (multiple AI assistants), the LLM can trigger transitions between team members via function calls. Currently, these transitions happen silently in Pipecat (Python) — no message is recorded in the conversation transcript, and no webhook event is fired. External systems and conversation reviewers have no visibility into when or why an assistant switch occurred.

## Goal

Record every team member switch as a notification message in the AI call transcript, with full audit details (from/to member info, AI config, transition trigger). The existing `aimessage_created` webhook event fires automatically, allowing external systems to react to switches in real-time.

## Design

### End-to-End Flow

```
Python transition handler (team_flow.py)
  | HTTP POST (fire-and-forget, async)
  v
Go pipecat-manager httphandler (POST /{id}/member-switched)
  | notifyHandler.PublishEvent()
  v
RabbitMQ (event: "team_member_switched")
  | subscribe
  v
Go ai-manager subscribehandler
  | messageHandler.EventPMTeamMemberSwitched()
  v
Create notification message in DB
  | (automatic)
  v
Webhook event: "aimessage_created" with role "notification"
```

### 1. New Message Role (bin-ai-manager)

Add `RoleNotification` to the message model:

```go
// models/message/main.go
RoleNotification Role = "notification"
```

### 2. Notification Message Content

The `Content` field of the notification message is a JSON string:

```json
{
    "type": "member_switched",
    "transition_function_name": "transfer_to_sales",
    "from_member": {
        "id": "uuid",
        "name": "Reception",
        "ai": {
            "engine_model": "openai.gpt-4o",
            "tts_type": "cartesia",
            "tts_voice_id": "voice-123",
            "stt_type": "deepgram"
        }
    },
    "to_member": {
        "id": "uuid",
        "name": "Sales Agent",
        "ai": {
            "engine_model": "openai.gpt-4o",
            "tts_type": "elevenlabs",
            "tts_voice_id": "voice-456",
            "stt_type": "deepgram"
        }
    }
}
```

Message metadata:
- `Role`: `"notification"`
- `Direction`: `"outgoing"` (system-generated, going into the transcript)

### 3. Python Side (bin-pipecat-manager/scripts/pipecat/team_flow.py)

**Track current member**: Use a mutable dict passed to all transition handlers.

```python
# In build_team_flow():
current_state = {"active_member_id": resolved_team["start_member_id"]}
```

**On transition**: After switching routing services, fire-and-forget an HTTP POST to Go.

```python
# In _create_transition_handler():
async def handler(args, flow_manager):
    from_member_id = current_state["active_member_id"]
    # ... existing: switch routing_llm, routing_tts, routing_stt ...
    current_state["active_member_id"] = next_member_id

    asyncio.create_task(_notify_member_switched(
        pipecatcall_id, from_member_id, next_member_id,
        function_name, resolved_team
    ))
    return {"status": "transferred"}, next_node
```

**`_notify_member_switched()`**: Extract member details from `resolved_team` dict (excluding `engine_key`), POST to Go HTTP endpoint. Errors are logged but do not block the transition.

### 4. New Event Type (bin-pipecat-manager)

```go
// models/message/event.go
EventTypeTeamMemberSwitched string = "team_member_switched"
```

New event payload structs in `models/message/`:

```go
type MemberSwitchedEvent struct {
    PipecatcallID            uuid.UUID  `json:"pipecatcall_id"`
    PipecatcallReferenceType string     `json:"pipecatcall_reference_type"`
    PipecatcallReferenceID   uuid.UUID  `json:"pipecatcall_reference_id"`
    TransitionFunctionName   string     `json:"transition_function_name"`
    FromMember               MemberInfo `json:"from_member"`
    ToMember                 MemberInfo `json:"to_member"`
}

type MemberInfo struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    EngineModel string    `json:"engine_model"`
    TTSType     string    `json:"tts_type"`
    TTSVoiceID  string    `json:"tts_voice_id"`
    STTType     string    `json:"stt_type"`
}
```

### 5. Go HTTP Endpoint (bin-pipecat-manager)

New route in `pkg/httphandler/main.go`:

```go
router.POST("/:id/member-switched", h.memberSwitchedHandle)
```

Handler delegates to `pipecatcallHandler.RunnerMemberSwitchedHandle()`.

### 6. Go pipecat-manager Handler

New method `RunnerMemberSwitchedHandle(id uuid.UUID, c *gin.Context)`:

1. Parse request body (from/to member details, transition function name)
2. Look up session to get `PipecatcallReferenceType` and `PipecatcallReferenceID`
3. Build `MemberSwitchedEvent` struct
4. Publish via `notifyHandler.PublishEvent(ctx, EventTypeTeamMemberSwitched, event)`

### 7. AI Manager Subscribe Handler

Subscribe to `team_member_switched` event from `pipecat-manager` publisher in `pkg/subscribehandler/main.go`.

New handler `processEventPMTeamMemberSwitched()` in `pkg/subscribehandler/pipecat_message.go`:

1. Deserialize event payload to pipecat-manager's `MemberSwitchedEvent`
2. Build notification content JSON with `from_member` and `to_member` objects
3. Call `messageHandler.EventPMTeamMemberSwitched(ctx, &evt)` to create the message

### 8. AI Manager Message Handler

New method `EventPMTeamMemberSwitched()` in `pkg/messagehandler/event.go`:

1. Find the AIcall ID from `PipecatcallReferenceID`
2. Build JSON content string
3. Call `h.Create(ctx, customerID, aicallID, DirectionOutgoing, RoleNotification, content, nil, "")`
4. The existing webhook event `aimessage_created` fires automatically

## Files Changed

| Service | File | Change |
|---------|------|--------|
| `bin-pipecat-manager` | `scripts/pipecat/team_flow.py` | Track current member, send HTTP notification on switch |
| `bin-pipecat-manager` | `models/message/event.go` | Add `EventTypeTeamMemberSwitched` |
| `bin-pipecat-manager` | `models/message/main.go` | Add `MemberSwitchedEvent` and `MemberInfo` structs |
| `bin-pipecat-manager` | `pkg/httphandler/main.go` | Add `POST /:id/member-switched` route |
| `bin-pipecat-manager` | `pkg/pipecatcallhandler/runner.go` | Add `RunnerMemberSwitchedHandle()` method and interface |
| `bin-ai-manager` | `models/message/main.go` | Add `RoleNotification` |
| `bin-ai-manager` | `pkg/subscribehandler/main.go` | Subscribe to `team_member_switched` event |
| `bin-ai-manager` | `pkg/subscribehandler/pipecat_message.go` | Add `processEventPMTeamMemberSwitched()` |
| `bin-ai-manager` | `pkg/messagehandler/event.go` | Add `EventPMTeamMemberSwitched()` handler |

## What Stays Unchanged

- **Webhook event type**: Reuses existing `aimessage_created` (fires automatically on message creation)
- **WebhookMessage**: Already includes `Role` field; consumers filter by `"notification"`
- **Database schema**: No migration needed (messages table accepts any role string)
- **Prometheus metrics**: `message_create_total` counter automatically tracks `notification` role

## Trade-offs

- **Fire-and-forget HTTP**: If the notification fails, the switch still happened but isn't recorded. Acceptable — the switch is the primary action, notification is secondary. Errors are logged.
- **Message ordering**: The notification is sent immediately after the switch but before the new LLM generates its first response. In practice, the notification will be recorded before the new member's first bot_llm message.
- **No new webhook event type**: Consumers must filter `aimessage_created` events by `role: "notification"`. This is simpler than adding a dedicated event type and keeps the webhook surface small.
