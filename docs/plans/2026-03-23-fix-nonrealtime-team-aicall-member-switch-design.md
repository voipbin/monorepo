# Fix Non-Realtime Team AIcall Member Switch

## Problem

When a non-realtime (message-based) AIcall uses a team, subsequent messages after a member switch use the wrong LLM. Each message creates an ephemeral Pipecat session using `AIEngineModel` from the AIcall record, which is always the start member's model.

Two root causes:

1. `EventPMTeamMemberSwitched` creates a notification message but never calls `UpdateCurrentMemberID()` — the AIcall record always points to the start member.
2. `SendReferenceTypeOthers()` uses `c.AIEngineModel` directly from the AIcall record without re-resolving the current member's AI config from the team.

Additionally, non-realtime AIcalls populate TTS/STT/VAD fields on the record even though there is no audio. This confuses the API response.

## Scope

All changes are in `bin-ai-manager` only. No cross-service API changes.

Known limitation: after a member switch, the original start member's system prompt remains in the message history. Tracked in #735.

## Design

### 1. Split startAIcall into startAIcallByRealtime / startAIcallByMessaging

Replace the single `startAIcall()` with two creation paths:

**`startAIcallByRealtime()`** — called by `startReferenceTypeCall`:
- Sets all AI fields: `AIEngineModel`, `AITTSType`, `AITTSVoiceID`, `AISTTType`, `AIVADConfig`, `AISmartTurnEnabled`
- Sets `ConfbridgeID`

**`startAIcallByMessaging()`** — called by `startReferenceTypeNone`, `startReferenceTypeConversation`:
- Sets only `AIEngineModel`
- TTS/STT/VAD fields left as zero values
- No confbridge

Both share common fields: `CustomerID`, `AssistanceType`, `AssistanceID`, `ActiveflowID`, `ReferenceType`, `ReferenceID`, `Gender`, `Language`, `Parameter`, `CurrentMemberID`, `Status`.

**API response impact**: Non-realtime AIcalls will return empty TTS/STT/VAD fields in the WebhookMessage. These fields were meaningless for non-realtime and only caused confusion.

### 2. Extract resolveTeamMemberAI shared helper

Extract the member-lookup-with-fallback pattern into a shared helper:

```go
func (h *aicallHandler) resolveTeamMemberAI(ctx context.Context, t *team.Team, memberID uuid.UUID) (*ai.AI, uuid.UUID, error)
```

Logic:
1. Search `t.Members` for `memberID`
2. If found, fetch that member's AI config via `aiHandler.Get(ctx, member.AIID)`
3. If not found, fall back to `t.StartMemberID` — search and fetch its AI config
4. If neither found, return error

Used by:
- `resolveAI()` (existing) — passes `team.StartMemberID`
- `SendReferenceTypeOthers()` (new) — passes `c.CurrentMemberID`

### 3. Update processEventPMTeamMemberSwitched to call UpdateCurrentMemberID

In `subscribehandler/pipecat_message.go`, `processEventPMTeamMemberSwitched()`:

After the existing call to `messageHandler.EventPMTeamMemberSwitched()`, add:

```go
h.aicallHandler.UpdateCurrentMemberID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)
```

`subscribeHandler` already has `aicallHandler` injected. `UpdateCurrentMemberID` already exists at db.go:176 with tests.

Error handling: log and continue. The notification message is already created, and the send path has a fallback to `StartMemberID`.

### 4. SendReferenceTypeOthers — fresh team resolution at send time

In `SendReferenceTypeOthers()`, before calling `startPipecatcall(ctx, c)`:

```
if c.AssistanceType == "team":
    1. Fetch team: teamHandler.Get(ctx, c.AssistanceID)
    2. Resolve member: resolveTeamMemberAI(ctx, team, c.CurrentMemberID)
    3. Override c.AIEngineModel with resolved AI's EngineModel
    4. If resolved memberID != c.CurrentMemberID:
       - Update CurrentMemberID on DB (fallback occurred)
```

For non-team AIcalls (`AssistanceType == "ai"`): no change, uses `c.AIEngineModel` as today.

Error handling: if team fetch or AI fetch fails, fall back to `c.AIEngineModel` and log a warning. Don't block the message send.

### 5. Tests

- `startAIcallByRealtime()`: verify TTS/STT/VAD fields are populated
- `startAIcallByMessaging()`: verify only AIEngineModel is set, TTS/STT/VAD are empty
- `processEventPMTeamMemberSwitched()`: verify UpdateCurrentMemberID is called with ToMember.ID
- `SendReferenceTypeOthers()` with team: verify team is fetched, correct AIEngineModel used
- `SendReferenceTypeOthers()` fallback: verify when CurrentMemberID not in team, falls back to StartMemberID and updates record
- Update existing tests that assert TTS/STT fields on non-realtime AIcalls

## Files Changed

| File | Change |
|------|--------|
| `pkg/aicallhandler/start.go` | Split `startAIcall` into two functions, extract `resolveTeamMemberAI`, refactor `resolveAI` to use it |
| `pkg/aicallhandler/send.go` | Add team resolution in `SendReferenceTypeOthers` |
| `pkg/aicallhandler/main.go` | Interface updates if needed |
| `pkg/subscribehandler/pipecat_message.go` | Call `UpdateCurrentMemberID` in `processEventPMTeamMemberSwitched` |
| `pkg/aicallhandler/*_test.go` | New and updated tests |
| `pkg/subscribehandler/*_test.go` | Updated test for member switched event |
