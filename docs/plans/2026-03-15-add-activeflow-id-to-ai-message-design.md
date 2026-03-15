# Design: Add activeflow_id to AI Message

## Problem

AI messages cannot be included in timeline-manager's aggregated events endpoint because they lack an `activeflow_id` field. The aggregated events endpoint filters by `activeflow_id` to combine events across all services for a single flow execution. Currently, AI messages only link to activeflows indirectly through the AIcall record (`Message.aicall_id` → `AIcall.activeflow_id`), which blocks direct event aggregation.

## Approach

Denormalize `activeflow_id` onto the AI message model, propagated through the Pipecat event pipeline. This follows the existing pattern where reference data (like `PipecatcallReferenceID`) flows from `Pipecatcall` through `Session` and event structs to the AI manager.

Data flow:
```
Pipecatcall.ActiveflowID
  → Session.ActiveflowID (new)
  → pmmessage.Message.ActiveflowID (new)
  → pmmessage.MemberSwitchedEvent.ActiveflowID (new)
  → ai-manager message.Message.ActiveflowID (new)
  → stored in ai_messages table (new column)
  → published via webhook event → timeline aggregation
```

For non-Pipecat message creation paths (RPC via `aicallHandler.Send`, tool execution, aicall start), the `activeflow_id` comes directly from the AIcall record which already has the field.

## Changes by Service

### bin-dbscheme-manager

New Alembic migration to add `activeflow_id` column to `ai_messages` table:
- `ALTER TABLE ai_messages ADD COLUMN activeflow_id binary(16)` (nullable)
- `CREATE INDEX idx_ai_messages_activeflow_id ON ai_messages(activeflow_id)`
- Downgrade: drop column

Must be deployed before code changes.

### bin-pipecat-manager

Models:
- `models/pipecatcall/session.go` — Add `ActiveflowID uuid.UUID` field to `Session` struct
- `models/message/main.go` — Add `ActiveflowID uuid.UUID` field to `Message` struct
- `models/message/member_switched.go` — Add `ActiveflowID uuid.UUID` field to `MemberSwitchedEvent` struct

Handlers:
- `pkg/pipecatcallhandler/session.go` — Populate `ActiveflowID: pc.ActiveflowID` in `SessionCreate()`
- `pkg/pipecatcallhandler/runner.go`:
  - `newMessageEvent()` — Populate `ActiveflowID: se.ActiveflowID` from session
  - `RunnerHTTPTeamMemberSwitched()` — Populate `ActiveflowID: pc.ActiveflowID` in `MemberSwitchedEvent`

### bin-ai-manager

Models:
- `models/message/main.go` — Add `ActiveflowID uuid.UUID` with `db:"activeflow_id,uuid"` tag
- `models/message/webhook.go` — Add `ActiveflowID uuid.UUID` to `WebhookMessage`; update `ConvertWebhookMessage()`
- `models/message/field.go` — Add `FieldActiveflowID Field = "activeflow_id"` constant

Interface + handler:
- `pkg/messagehandler/main.go` — Add `activeflowID uuid.UUID` parameter to `Create()` in interface
- `pkg/messagehandler/db.go` — Accept and set `activeflowID` in `Create()` implementation

All callers of `messageHandler.Create()` (6 call sites):
- `pkg/messagehandler/event.go` (lines 23, 42, 56, 101) — Pass `evt.ActiveflowID`
- `pkg/aicallhandler/send.go` (lines 40, 63) — Pass `c.ActiveflowID`
- `pkg/aicallhandler/tool.go` (lines 38, 100) — Pass `c.ActiveflowID`
- `pkg/aicallhandler/start.go` (lines 178, 417) — Pass `res.ActiveflowID` / `c.ActiveflowID`

### bin-openapi-manager

- `openapi/openapi.yaml` — Add `activeflow_id` property to `AIManagerMessage` schema:
  ```yaml
  activeflow_id:
    type: string
    format: uuid
    x-go-type: string
    description: "The unique identifier of the active flow. Returned from the `POST /activeflows` response."
    example: "550e8400-e29b-41d4-a716-446655440000"
  ```
- Regenerate: `go generate ./...`

### bin-api-manager

- Regenerate server code: `go generate ./...`

## What's NOT Changing

- **bin-timeline-manager** — Already filters by `activeflow_id` on events; no changes needed
- **dbhandler** — Generic filter infrastructure (`commondatabasehandler.ApplyFields`) handles new fields automatically
- **listenhandler** — RPC message creation goes through `aicallHandler.Send()` which has the AIcall record

## Deployment Order

1. Alembic migration (add column to database)
2. Code deployment (all services)

## Test Updates

- `bin-ai-manager/pkg/messagehandler/event_test.go` — Update mock expectations for new `Create()` signature
- `bin-ai-manager/pkg/messagehandler/db_test.go` — Add `activeflowID` parameter to test cases
- `bin-ai-manager/pkg/aicallhandler/send_test.go` — Update mock expectations
- `bin-ai-manager/pkg/aicallhandler/tool_test.go` — Update mock expectations
- `bin-ai-manager/pkg/aicallhandler/start_test.go` — Update mock expectations
- `bin-pipecat-manager/pkg/pipecatcallhandler/session_test.go` — Verify `ActiveflowID` populated
- `bin-pipecat-manager/pkg/pipecatcallhandler/runner_test.go` — Update `newMessageEvent` tests
- Mock regeneration via `go generate ./...` in both services
