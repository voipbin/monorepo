# Design: Add `active_ai_id` to AI Messages

**Date:** 2026-05-21
**Branch:** NOJIRA-Add-active-ai-id-to-aimessage
**Status:** Approved

## Background

Every AI message (`ai_messages` table) is associated with an AIcall, but there is currently no field recording which specific AI configuration was active when the message was created. For simple (non-team) AIcalls, this is `aicall.AssistanceID`. For team AIcalls, the active AI changes as team members switch — `aicall.CurrentMemberID` points to the current team member, and each team member has an `AIID` that identifies the actual AI config.

Consumers of the `aimessage_created` and `aimessage_intermediate` webhook events currently cannot determine which AI spoke or was listening without making additional API calls.

## Goal

Add an `active_ai_id` field to `message.Message`, `WebhookMessage`, and `IntermediateWebhookMessage` that always contains the UUID of the AI config that was active at the moment the message was created.

- For assistant messages: the AI that generated the response.
- For user messages: the AI that was listening.
- For system, tool, and notification messages: the AI that was active at creation time.

## Scope

Changes are contained entirely within `bin-ai-manager`. No changes to `bin-pipecat-manager`, `bin-conversation-manager`, or any other service.

## Design

### 1. Data model (`bin-ai-manager/models/message/`)

**`main.go`** — add one field to `Message`:
```go
ActiveAIID uuid.UUID `json:"active_ai_id,omitempty" db:"active_ai_id,uuid"`
```

**`field.go`** — add constant for DB queries and filters:
```go
FieldActiveAIID Field = "active_ai_id"
```

**`webhook.go`** — add to `WebhookMessage`, `IntermediateWebhookMessage`, and propagate in `ConvertWebhookMessage()`:
```go
ActiveAIID uuid.UUID `json:"active_ai_id,omitempty"`
```

### 2. Database migration (`bin-dbscheme-manager`)

One Alembic migration on the `ai_messages` table:

```sql
-- upgrade
ALTER TABLE ai_messages
  ADD COLUMN active_ai_id BINARY(16) NOT NULL DEFAULT 0x00000000000000000000000000000000;

-- downgrade
ALTER TABLE ai_messages DROP COLUMN active_ai_id;
```

Non-nullable with zero-UUID default. Historical rows will have `Nil`; all new rows will carry the real AI ID going forward.

### 3. Resolution helper + `CreateOption` (`bin-ai-manager/pkg/messagehandler/`)

**`main.go`** additions:
- Add `activeAIID uuid.UUID` to `createParams`
- Add `WithActiveAIID(id uuid.UUID) CreateOption`
- Set `m.ActiveAIID = p.activeAIID` inside `Create()`

**New private helper `resolveActiveAIID`** — resolves the active AI UUID for a given AIcall. Non-blocking: logs a warning and returns `uuid.Nil` on any error so message creation continues uninterrupted.

```
resolveActiveAIID(ctx, aicallID) uuid.UUID:
  1. Fetch AIcall via h.reqHandler.AIV1AIcallGet(ctx, aicallID)
     → on error: log + return uuid.Nil
  2. Switch on aicall.AssistanceType:
     - AssistanceTypeAI:
         return aicall.AssistanceID
     - AssistanceTypeTeam:
         t := h.db.TeamGet(ctx, aicall.AssistanceID)
         → on error: log + return uuid.Nil
         for m in t.Members:
           if m.ID == aicall.CurrentMemberID: return m.AIID
         → not found: log + return uuid.Nil
     - default: return uuid.Nil
```

`TeamGet` is available on the existing `dbhandler.DBHandler` interface (backed by Redis cache), so no new dependencies are introduced.

### 4. Caller updates

#### `messagehandler/event.go`

Each handler resolves `active_ai_id` and passes it via `WithActiveAIID`:

| Handler | Resolution source |
|---|---|
| `EventPMMessageUserTranscription` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` |
| `EventPMMessageBotLLM` | AIcall already fetched for conversation path; resolve from it. Add fetch for voice/task path. |
| `EventPMMessageBotLLMIntermediate` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)`; set directly on `IntermediateWebhookMessage.ActiveAIID` (no DB write). |
| `EventPMMessageUserLLM` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` |
| `EventPMTeamMemberSwitched` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` — `MemberInfo` carries no `AIID`; rely on `CurrentMemberID` having been updated before the event is processed. |
| `EventPMPipecatcallTerminated` | AIcall already fetched; resolve from it (same as BotLLM conversation path). |

#### `aicallhandler/` (`start.go`, `send.go`, `tool.go`)

All `h.messageHandler.Create(...)` call sites in `aicallhandler` already have the AI config `c` in scope (returned by `resolveAI`). Pass `WithActiveAIID(c.ID)` at each call site:

- `start.go:230` — user message created at call start
- `start.go:475` — system message (init prompt)
- `send.go:47` — user message from explicit send
- `send.go:70` — user message from terminate-with-send path
- `tool.go:40` — assistant message wrapping a tool call
- `tool.go:103` — tool result message

### 5. RST documentation (`bin-api-manager/docsdev/source/`)

Update the aimessage struct documentation to include `active_ai_id` with a description. Rebuild HTML after editing.

## Invariants

- `active_ai_id` is always the UUID of an `ai` resource (never a team UUID).
- `active_ai_id` may be `uuid.Nil` for historical rows (before this migration) and for any future edge cases where resolution fails.
- The field is present on all message roles (`user`, `assistant`, `system`, `tool`, `notification`).
- The field is present in both `aimessage_created` and `aimessage_intermediate` webhook events.

## Out of scope

- Adding `active_ai_id` to AIcall itself (Approach B considered and rejected — unnecessary broader change).
- Propagating AI ID through pipecat-manager events (Approach C considered and rejected — out of scope for this change).
- Backfilling historical rows — left as `uuid.Nil`.
