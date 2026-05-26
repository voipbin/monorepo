# AIcall Metadata Field + Prompt Version Tracking — Design Spec

**Date:** 2026-05-26
**Branch:** NOJIRA-Add-aicall-metadata-prompt-version-tracking
**Status:** Draft

---

## Background

This design is a prerequisite for the VoIPBin Assistants Audit feature. The audit feature will evaluate whether an AI Assistant followed its prompt instructions during an AIcall. Before audit can be implemented, the following foundational data must be captured at call start time.

---

## Problem Statement

Three gaps exist in the current system:

1. **Team AIcall prompt gap.** At call start, only the start member's `init_prompt` is saved as a `role=system` message in `ai_messages`. All other team members' prompts exist only in pipecat runner memory and are lost when the call ends. There is no way to recover what prompt a non-start team member was running on.

2. **No prompt version on AIcall.** `AIcall` has no field recording which version of `init_prompt` was active at call start. `ai_ai_prompt_histories` records changes over time, but the AIcall record itself has no pointer to the version it used.

3. **No current version pointer on AI.** `AI` stores the current `init_prompt` text but has no field indicating which `prompt_history` entry that corresponds to. Callers must query `GET /ais/{id}/prompt_histories?page_size=1` as an extra round-trip.

---

## Goals

- Capture the **final, variable-substituted prompt** for every AI participant at AIcall start time, stored on the AIcall record.
- Add a **version pointer** (`current_prompt_history_id`) to the `AI` model, maintained atomically on every `init_prompt` change.
- Expose this data to customers via **webhook events**.
- Use a **generic `Metadata` field** on AIcall so future audit features can store additional data without schema migrations.

## Non-Goals

- Changing how pipecat receives prompts.
- Backfilling `current_prompt_history_id` for existing AIs (they receive a zero UUID; the history table is authoritative for historical lookups).
- Implementing the audit scoring feature itself.

---

## Data Model Changes

### `ai_ais` table — new column

```sql
current_prompt_history_id BINARY(16) NOT NULL DEFAULT (X'00000000000000000000000000000000')
```

- Zero UUID (`00000000-0000-0000-0000-000000000000`) means no versioned history has been recorded yet for this AI.
- Updated atomically alongside every `init_prompt` write (create or update).

### `ai_aicalls` table — new column

```sql
metadata JSON NOT NULL DEFAULT '{}'
```

- Generic key-value store. Prompt snapshots are stored under the key `"prompt_snapshots"`.
- Additional audit or operational data can be added under new keys in the future without schema changes.

### Go model — `models/ai/main.go`

```go
CurrentPromptHistoryID uuid.UUID `json:"current_prompt_history_id,omitempty" db:"current_prompt_history_id,uuid"`
```

### Go model — `models/aicall/main.go`

```go
// PromptSnapshot records the prompt version and final substituted text for one
// AI participant at AIcall start time.
type PromptSnapshot struct {
    AIID            uuid.UUID `json:"ai_id"`
    PromptHistoryID uuid.UUID `json:"prompt_history_id,omitempty"` // zero UUID ("00000000-...") means no history recorded yet
    Prompt          string    `json:"prompt"`                      // variable-substituted final prompt
    MemberID        uuid.UUID `json:"member_id,omitempty"`         // team AIcall only
}
// Note: uuid.UUID is [16]byte; Go's omitempty does not omit zero-value fixed-size arrays.
// Consumers must check for "00000000-0000-0000-0000-000000000000" to detect "no history".
// This matches the existing codebase convention (e.g. ConfbridgeID in webhook.go).

// MetaKeyPromptSnapshots is the metadata key for prompt snapshots.
// Always use this constant instead of the raw string.
const MetaKeyPromptSnapshots = "prompt_snapshots"

type AIcall struct {
    // ... existing fields unchanged ...
    Metadata map[string]any `json:"metadata,omitempty" db:"metadata,json"`
}
```

### Go model — `models/aicall/webhook.go`

`Metadata` is added to `WebhookMessage` and `ConvertWebhookMessage()` so customers receive prompt snapshots in every `aicall.*` event.

```go
type WebhookMessage struct {
    // ... existing fields unchanged ...
    Metadata map[string]any `json:"metadata,omitempty"`
}
```

---

## Handler Changes

### A. AI create/update path — maintain `current_prompt_history_id`

The key principle: **pre-generate both IDs before any write**, write the AI row first (so it holds the pointer from the start), then write the history row.

**AI Create (`aihandler.Create`):**

```
aiID      := utilHandler.UUIDCreate()   // pre-generate AI ID
historyID := utilHandler.UUIDCreate()   // pre-generate history ID

if initPrompt != "" {
    1. dbCreate(ctx, ..., ID: aiID, CurrentPromptHistoryID: historyID, InitPrompt: initPrompt)
    2. AIPromptHistoryCreate(ctx, {ID: historyID, AIID: aiID, CustomerID: ..., Prompt: initPrompt})
} else {
    dbCreate(ctx, ..., ID: aiID, CurrentPromptHistoryID: uuid.Nil, InitPrompt: "")
    // no history row
}
```

**AI Update (`aihandler.Update`) — prompt changed:**

```
historyID := utilHandler.UUIDCreate()   // pre-generate history ID

1. dbUpdate(ctx, id, {..., InitPrompt: newPrompt, CurrentPromptHistoryID: historyID})
2. AIPromptHistoryCreate(ctx, {ID: historyID, AIID: id, CustomerID: ..., Prompt: newPrompt})
```

**AI Update — prompt unchanged:** `dbUpdate` only; `current_prompt_history_id` untouched.

**Error handling:** `AIPromptHistoryCreate` failures are non-fatal (consistent with the existing pattern). Log and continue. If it fails, `current_prompt_history_id` points to a non-existent row; the AI's `init_prompt` remains the source of truth.

---

### B. New `resolveAIForTeam()` in `aicallhandler`

```go
// resolveAIForTeam fetches all team members' AI configs, keyed by member ID.
// Used at AIcall start to build prompt snapshots for every team member.
func (h *aicallHandler) resolveAIForTeam(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]*ai.AI, error)
```

- Calls `teamHandler.Get(ctx, teamID)` to get the team and its `Members` slice.
- Fetches each member's AI via `aiHandler.Get(ctx, m.AIID)` in parallel (goroutines + `sync.WaitGroup`).
- Returns `map[memberID → *ai.AI]` and an error.
- **Partial failure handling**: if one or more member AI fetches fail, log a warning for each failure and return the partial map (excluding failed members) with a nil error. The call start continues with whatever snapshots were collectible. A total failure (e.g. `teamHandler.Get` itself fails) returns an error and aborts call start.
- Called only for `AssistanceTypeTeam`.

---

### C. Build `[]PromptSnapshot` at call start

In both `startAIcallByRealtime()` and `startAIcallByMessaging()`, after `resolveAI()` returns and before `Create()`:

**Single-AI call (`AssistanceTypeAI`):**

```go
substitutedPrompt := h.getInitPrompt(ctx, a, activeflowID)
snapshots := []aicall.PromptSnapshot{
    {
        AIID:            a.ID,
        PromptHistoryID: a.CurrentPromptHistoryID, // zero UUID → omitted in JSON
        Prompt:          substitutedPrompt,
    },
}
```

**Team call (`AssistanceTypeTeam`):**

```go
memberAIs, err := h.resolveAIForTeam(ctx, assistanceID)
// err → return error

snapshots := make([]aicall.PromptSnapshot, 0, len(memberAIs))
for memberID, memberAI := range memberAIs {
    substitutedPrompt := h.getInitPrompt(ctx, memberAI, activeflowID)
    snapshots = append(snapshots, aicall.PromptSnapshot{
        AIID:            memberAI.ID,
        PromptHistoryID: memberAI.CurrentPromptHistoryID,
        Prompt:          substitutedPrompt,
        MemberID:        memberID,
    })
}
```

In both cases:

```go
metadata := map[string]any{
    aicall.MetaKeyPromptSnapshots: snapshots,
}
```

Note: `getInitPrompt()` is called here for each AI, and called again inside `startInitMessages()` for the start member — two RPC calls to `FlowV1VariableSubstitute` for that AI. Acceptable now; can be eliminated in a later refactor by passing the substituted prompt into `startInitMessages()`.

---

### D. `Create()` / `CreateByMessaging()` — accept `metadata`

Both functions gain a `metadata map[string]any` parameter included in the SQL INSERT. The `Metadata` field on the returned `AIcall` is populated from the written value.

Callers: only `startAIcallByRealtime()` and `startAIcallByMessaging()` — both already being modified in this change.

---

## Alembic Migrations

Two migrations required, in `bin-dbscheme-manager/bin-manager/main/versions/`. Always generate with `alembic revision -m "..."` — never hand-author revision IDs.

**Migration 1 — `ai_ais` column:**
```sql
-- upgrade
ALTER TABLE ai_ais
    ADD COLUMN current_prompt_history_id BINARY(16) NOT NULL
    DEFAULT (X'00000000000000000000000000000000');

-- downgrade
ALTER TABLE ai_ais DROP COLUMN current_prompt_history_id;
```

**Migration 2 — `ai_aicalls` column:**
```sql
-- upgrade
ALTER TABLE ai_aicalls
    ADD COLUMN metadata JSON NOT NULL DEFAULT (JSON_OBJECT());

-- downgrade
ALTER TABLE ai_aicalls DROP COLUMN metadata;
```

Migration 2 depends on Migration 1 (`down_revision` chains correctly via `alembic revision`).

---

## API / OpenAPI

- `GET /ais/{id}` response: expose `current_prompt_history_id`.
- `GET /aicalls/{id}` response: expose `metadata` (customers see `prompt_snapshots` under it).
- Webhook events (`aicall.start`, `aicall.end`, etc.): expose `metadata` via `WebhookMessage`.
- No new endpoints required.

---

## Testing

- `aihandler`: table-driven tests for `Create` and `Update` covering:
  - Prompt set on create → history row created, `current_prompt_history_id` matches
  - Prompt changed on update → new history row, ID updated
  - Prompt unchanged on update → no history row, ID untouched
  - `AIPromptHistoryCreate` failure → non-fatal, AI update still returns success
- `aicallhandler`: extend `startAIcallByRealtime` and `startAIcallByMessaging` tests:
  - Single-AI call → `Metadata` contains one `PromptSnapshot` with correct fields
  - Team call → `Metadata` contains one `PromptSnapshot` per member, all with `MemberID` set
  - `resolveAIForTeam` → unit test for parallel fetch and map construction
- Existing tests must continue to pass (no regression).

---

## Implementation Order

1. Alembic migrations (schema first)
2. `models/ai/main.go` — add `CurrentPromptHistoryID`
3. `models/aicall/main.go` — add `Metadata`, `PromptSnapshot`, `MetaKeyPromptSnapshots`
4. `models/aicall/webhook.go` — expose `Metadata`
5. `pkg/dbhandler/` — `AICreate` / `AIUpdate` accept `current_prompt_history_id`; `AIcallCreate` accepts `metadata`
6. `pkg/aihandler/` — update `Create` and `Update` write sequence
7. `pkg/aicallhandler/` — add `resolveAIForTeam`, update `startAIcallByRealtime` / `startAIcallByMessaging` / `Create` / `CreateByMessaging`
8. OpenAPI spec update (`bin-api-manager`)
9. Tests
10. Full verification: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
