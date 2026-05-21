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

**New private helper `resolveActiveAIID`** — resolves the active AI UUID for a given AIcall ID. Non-blocking: logs a `Warnf` and returns `uuid.Nil` on any error so message creation continues uninterrupted.

```
resolveActiveAIID(ctx, aicallID) uuid.UUID:
  1. Fetch AIcall via h.reqHandler.AIV1AIcallGet(ctx, aicallID)
     → on error: logrus.Warnf + return uuid.Nil
  2. Switch on aicall.AssistanceType:
     - AssistanceTypeAI:
         return aicall.AssistanceID
     - AssistanceTypeTeam:
         t := h.db.TeamGet(ctx, aicall.AssistanceID)
         → on error: logrus.Warnf + return uuid.Nil
         for m in t.Members:
           if m.ID == aicall.CurrentMemberID: return m.AIID
         → not found: logrus.Warnf + return uuid.Nil
     - default: return uuid.Nil
```

**New private helper `resolveTeamMemberAIID`** — for the `EventPMTeamMemberSwitched` case where the notification message is created *before* `UpdateCurrentMemberID` commits, so `resolveActiveAIID` (which reads `CurrentMemberID`) would return the outgoing member's AI ID instead of the incoming one. This helper takes the explicit incoming member ID.

```
resolveTeamMemberAIID(ctx, aicallID, memberID uuid.UUID) uuid.UUID:
  1. Fetch AIcall via h.reqHandler.AIV1AIcallGet(ctx, aicallID)
     → on error: logrus.Warnf + return uuid.Nil
  2. If aicall.AssistanceType != AssistanceTypeTeam: return uuid.Nil
  3. t := h.db.TeamGet(ctx, aicall.AssistanceID)
     → on error: logrus.Warnf + return uuid.Nil
  4. for m in t.Members:
       if m.ID == memberID: return m.AIID
     → not found: logrus.Warnf + return uuid.Nil
```

`TeamGet` is available on the existing `dbhandler.DBHandler` interface (backed by Redis cache), so no new dependencies are introduced.

### 4. Caller updates

#### `messagehandler/event.go`

Each handler resolves `active_ai_id` and passes it via `WithActiveAIID`:

| Handler | Resolution source | Notes |
|---|---|---|
| `EventPMMessageUserTranscription` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` | |
| `EventPMMessageBotLLM` — conversation path | Inline `resolveActiveAIIDFromAIcall(ctx, ac)` or equivalent switch on `ac.AssistanceType` using the already-fetched `ac` — **do not** call `resolveActiveAIID` again as that doubles the RPC | AIcall fetch already present at line 83 |
| `EventPMMessageBotLLM` — voice/task path (`ReferenceTypeAICall`) | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` | Requires adding AIcall fetch on this path |
| `EventPMMessageBotLLM` — non-AICall path (early guard, line ~73) | `uuid.Nil` explicitly | `evt.PipecatcallReferenceID` is not an aicall ID here; resolution is impossible |
| `EventPMMessageBotLLMIntermediate` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)`; set on `IntermediateWebhookMessage.ActiveAIID` directly | No DB write |
| `EventPMMessageUserLLM` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` | |
| `EventPMTeamMemberSwitched` | `resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)` | Cannot use `resolveActiveAIID` — notification message is created **before** `UpdateCurrentMemberID` commits, so `CurrentMemberID` still reflects the outgoing member |
| `EventPMPipecatcallTerminated` | Resolve from already-fetched AIcall (`ac`) | AIcall fetch already present |

#### `aicallhandler/` (`start.go`, `send.go`, `tool.go`)

In `start.go`, the resolved `*ai.AI` config `a` is in scope at line 475 (init prompt). At line 230 (`startReferenceTypeConversation`), `a` reflects the start member and may not represent the current active AI for a resumed team session — use `resolveActiveAIID` there instead.

In `send.go` and `tool.go`, `c` is `*aicall.AIcall` (not `*ai.AI`), so `c.ID` is the **aicall UUID** — `WithActiveAIID(c.ID)` would store the wrong type. Use `resolveActiveAIID(ctx, c.ID)` at all these sites.

| Call site | Resolution |
|---|---|
| `start.go:475` — system message (init prompt) | `WithActiveAIID(a.ID)` — `a *ai.AI` is the resolved AI config, always correct at creation time |
| `start.go:230` — user message at conversation start | `resolveActiveAIID(ctx, res.ID)` — handles resumed team sessions where `CurrentMemberID` ≠ start member |
| `send.go:47` — user message from explicit send | `resolveActiveAIID(ctx, c.ID)` — `c` is `*aicall.AIcall` |
| `send.go:70` — user message from terminate-with-send path | `resolveActiveAIID(ctx, aicallID)` — aicall ID is the parameter |
| `tool.go:40` — assistant message wrapping a tool call | `resolveActiveAIID(ctx, c.ID)` — `c` is `*aicall.AIcall` |
| `tool.go:103` — tool result message | `resolveActiveAIID(ctx, c.ID)` — `c` is `*aicall.AIcall` |

### 5. Service domain docs (`bin-ai-manager/docs/domain.md`)

The monorepo `check-service-docs.sh` hook warns when `models/.../*.go` changes without a matching `docs/domain.md` update. Add a note to the Message section listing `active_ai_id` and its semantics. This suppresses the hook warning and keeps the service's internal documentation accurate.

### 6. RST documentation (`bin-api-manager/docsdev/source/`)

Update `ai_struct_message.rst` to include `active_ai_id` with a description. After editing, perform a clean rebuild and force-add the built HTML (as required by the monorepo CLAUDE.md):

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```

Commit the RST source and built HTML together in the same commit as the Go changes.

## Invariants

- `active_ai_id` is always the UUID of an `ai` resource (never a team UUID).
- `active_ai_id` may be `uuid.Nil` for historical rows (before this migration), for the non-AICall early-guard path in `EventPMMessageBotLLM`, and for any future edge cases where resolution fails. When `uuid.Nil`, the field serialises as `"00000000-0000-0000-0000-000000000000"` in webhook JSON — `uuid.UUID` is `[16]byte`, and Go's `encoding/json` does not suppress array types via `omitempty`. Consumers should treat the zero UUID string as equivalent to no active AI being identifiable.
- The field is present on all message roles (`user`, `assistant`, `system`, `tool`, `notification`).
- The field is present in both `aimessage_created` and `aimessage_intermediate` webhook events.
- All error paths in `resolveActiveAIID` and `resolveTeamMemberAIID` log at `Warnf` level (not `Errorf`) since `uuid.Nil` is an accepted degraded outcome, not a failure.
- `FieldActiveAIID` is added for completeness but filtering `MessageList` by `active_ai_id` is **not** added in this scope — no API change to the list endpoint.
- Mock regeneration: `WithActiveAIID` and the new helpers are unexported and do not change the `MessageHandler` interface, so the generated mock file does not need updating. However, `go generate ./...` is still a **mandatory step** in the monorepo verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`) and must not be skipped.
- `EventPMMessageUserLLM` has no `PipecatcallReferenceType` guard (unlike some sibling handlers). If it is ever invoked for a non-AICall pipecatcall, `resolveActiveAIID` will fail silently with a `Warnf` and `uuid.Nil`. This is an accepted consequence of the non-blocking design and is not introduced by this change.

## Out of scope

- Adding `active_ai_id` to AIcall itself (Approach B considered and rejected — unnecessary broader change).
- Propagating AI ID through pipecat-manager events (Approach C considered and rejected — out of scope for this change).
- Backfilling historical rows — left as `uuid.Nil`.
- Filtering `MessageList` by `active_ai_id` — deferred to a future change if needed.
