# AIcall Metadata Field + Prompt Version Tracking — Design Spec

**Date:** 2026-05-26
**Branch:** NOJIRA-Add-aicall-metadata-prompt-version-tracking
**Status:** Approved (5-round review, 2026-05-26)

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
CurrentPromptHistoryID uuid.UUID `json:"current_prompt_history_id" db:"current_prompt_history_id,uuid"`
```

Note: `omitempty` is intentionally absent — `uuid.UUID` is `[16]byte` and Go's std JSON encoder does not omit zero-value fixed-size arrays. The zero UUID is the valid "no history" sentinel.

### Go model — `models/ai/field.go`

Add the new field constant so `AIUpdate` field-map calls can reference it:

```go
FieldCurrentPromptHistoryID Field = "current_prompt_history_id"
```

### Go model — `models/aicall/main.go`

```go
// PromptSnapshot records the prompt version and final substituted text for one
// AI participant at AIcall start time.
type PromptSnapshot struct {
    AIID            uuid.UUID `json:"ai_id"`
    PromptHistoryID uuid.UUID `json:"prompt_history_id"` // zero UUID ("00000000-...") means no history recorded yet
    Prompt          string    `json:"prompt"`            // variable-substituted final prompt; raw text when activeflowID is nil
    MemberID        uuid.UUID `json:"member_id"`         // zero UUID for single-AI calls; member UUID for team calls
}
// Note: uuid.UUID is [16]byte; Go's omitempty does NOT omit zero-value fixed-size arrays,
// so omitempty is intentionally absent. Consumers must check for
// "00000000-0000-0000-0000-000000000000" to detect "no history" or "no member".
// This matches the existing codebase convention (e.g. ConfbridgeID in webhook.go).

// MetaKeyPromptSnapshots is the metadata key for prompt snapshots.
// Always use this constant instead of the raw string.
const MetaKeyPromptSnapshots = "prompt_snapshots"

type AIcall struct {
    // ... existing fields unchanged ...
    Metadata map[string]any `json:"metadata,omitempty" db:"metadata,json"`
}
```

### Go model — `models/aicall/field.go`

Add the field constant for future update calls and test mock setup:

```go
FieldMetadata Field = "metadata"
```

**JSON scanning note:** The DB column defaults to `'{}'` / `JSON_OBJECT()`. When scanned via `commondatabasehandler.ScanRow` with `db:"metadata,json"`, an empty JSON object scans as an empty (but non-nil) `map[string]any{}`. In Go's `encoding/json`, `omitempty` only omits a map field when the map is **nil** — a non-nil empty map is not omitted. Therefore, pre-migration AIcalls (and any AIcall with no metadata written) will always produce `"metadata": {}` in API and webhook JSON output, never an absent field. This is the intended behavior — the field is always present.

```go
// remainder of models/aicall/main.go unchanged
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

`dbCreate` generates the AI's own ID internally. To carry `CurrentPromptHistoryID` into `dbCreate`, add `currentPromptHistoryID uuid.UUID` as a new parameter to `dbCreate` (it is a private method; its only callers are within `aihandler`). `dbCreate` sets the field on the `ai.AI` struct before calling `h.db.AICreate`.

```
if initPrompt != "" {
    historyID := utilHandler.UUIDCreate()   // pre-generate history ID
    1. result := dbCreate(ctx, ..., currentPromptHistoryID: historyID, InitPrompt: initPrompt)
       // dbCreate generates result.ID internally; writes row with CurrentPromptHistoryID = historyID
    2. AIPromptHistoryCreate(ctx, {ID: historyID, AIID: result.ID, CustomerID: ..., Prompt: initPrompt})
} else {
    dbCreate(ctx, ..., currentPromptHistoryID: uuid.Nil, InitPrompt: "")
    // no history row
}
```

**AI Update (`aihandler.Update`) — restructuring required:**

The current `Update()` calls `dbUpdate` unconditionally for all cases. This must be restructured: the changed and cleared branches **replace** `dbUpdate` entirely with `h.db.AIUpdate` (the field-map variant), so that `current_prompt_history_id` is written atomically with `init_prompt` in a single DB call. `dbUpdate` is only used for the unchanged branch.

**Branch: prompt changed to non-empty —** do NOT call `dbUpdate`; call `h.db.AIUpdate` directly:

```
historyID := utilHandler.UUIDCreate()   // pre-generate history ID

1. h.db.AIUpdate(ctx, id, map[ai.Field]any{
       // include all fields that dbUpdate writes (name, detail, engine model, etc.)
       ai.FieldInitPrompt:             newPrompt,
       ai.FieldCurrentPromptHistoryID: historyID,
   })
2. AIPromptHistoryCreate(ctx, {ID: historyID, AIID: id, CustomerID: ..., Prompt: newPrompt})
```

**Branch: prompt explicitly cleared to empty string (`newPrompt == ""` AND `preUpdateAI.InitPrompt != ""`) —** requires a pre-fetch of the current AI to distinguish from "unchanged-empty" (see below). Do NOT call `dbUpdate`; call `h.db.AIUpdate` directly. The pre-fetch result can be shared with the changed branch (fetch once, use across all three branches):

```
h.db.AIUpdate(ctx, id, map[ai.Field]any{
    // include all fields that dbUpdate writes
    ai.FieldInitPrompt:             "",
    ai.FieldCurrentPromptHistoryID: uuid.Nil,
})
// no history row; CurrentPromptHistoryID reset to zero UUID
```

**Branch: prompt unchanged** (`newPrompt == ""` AND `preUpdateAI.InitPrompt == ""`, or `newPrompt == preUpdateAI.InitPrompt`) — use the existing `dbUpdate` wrapper as-is; `current_prompt_history_id` is not written (untouched in DB).

**Error handling:** `AIPromptHistoryCreate` failures are non-fatal (consistent with the existing pattern). Log and continue. If it fails, `current_prompt_history_id` points to a non-existent row; the AI's `init_prompt` remains the source of truth.

---

### B. New `resolveAIForTeam()` in `aicallhandler`

`resolveAI()` is **not** modified. Its existing four-value return signature — `(*ai.AI, map[string]any, uuid.UUID, error)` (returning start-member AI, team parameter map, currentMemberID, error) — is preserved to avoid breaking the call sites in `Start()` and `StartTask()`.

`resolveAIForTeam()` is a new, standalone helper that takes the team ID and fetches the team independently:

```go
// resolveAIForTeam fetches all team members' AI configs, keyed by member ID.
// It calls teamHandler.Get once internally. For team calls this means teamHandler.Get
// is called twice total (once inside resolveAI, once here) — an acceptable trade-off
// that can be eliminated in a later refactor.
func (h *aicallHandler) resolveAIForTeam(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]*ai.AI, error)
```

- Calls `teamHandler.Get(ctx, teamID)` to get the team and its `Members` slice.
- Fetches each member's AI via `aiHandler.Get(ctx, m.AIID)` in parallel (goroutines + `sync.WaitGroup`).
- **Concurrency-safe map construction**: use a `sync.Mutex`-protected `map[uuid.UUID]*ai.AI` (or equivalent channel-collect pattern) to safely accumulate results from goroutines. A data race on a plain map is not acceptable.
- Returns `map[memberID → *ai.AI]` and an error.
- **No `StartMemberID` fallback**: unlike `resolveTeamMemberAI()`, this function fetches every member by its explicit `AIID` and does not fall back to a default member AI.
- **Partial failure handling**: if one or more member AI fetches fail, log a warning for each failure and return the partial map (excluding failed members) with a nil error. The call start continues with whatever snapshots were collectible. A total failure (e.g. `teamHandler.Get` itself fails) returns an error and aborts call start.
- **All-members-fail case**: if every member AI fetch fails, `resolveAIForTeam` returns an empty (non-nil) map and nil error. The caller builds an empty `[]PromptSnapshot{}`, and `Create()` proceeds with `metadata: {"prompt_snapshots": []}`. This is accepted behavior — call creation is not blocked by snapshot collection failures.
- Called only for `AssistanceTypeTeam`.

---

### C. Build `[]PromptSnapshot` at call start

In both `startAIcallByRealtime()` and `startAIcallByMessaging()`, after `resolveAI()` returns and before `Create()`. `activeflowID` used for snapshot building is the same `activeflowID` value passed to `Create()` — do not read it from the returned AIcall record post-create.

**`activeflowID == uuid.Nil` behavior:** When `activeflowID` is nil (e.g. in `startReferenceTypeNone`), `getInitPrompt()` returns the raw, un-substituted `init_prompt` text without calling `FlowV1VariableSubstitute`. The `PromptSnapshot.Prompt` field stores this raw text. This is the intended behavior — the snapshot captures whatever prompt was actually in effect.

**Single-AI call (`AssistanceTypeAI`):**

```go
// a, _, _, err := h.resolveAI(ctx, assistanceType, assistanceID)
substitutedPrompt := h.getInitPrompt(ctx, a, activeflowID)
snapshots := []aicall.PromptSnapshot{
    {
        AIID:            a.ID,
        PromptHistoryID: a.CurrentPromptHistoryID,
        Prompt:          substitutedPrompt,
        // MemberID: uuid.Nil (zero value, serialized as "00000000-...")
    },
}
```

**Team call (`AssistanceTypeTeam`):**

```go
// a, _, _, err := h.resolveAI(ctx, assistanceType, assistanceID)
memberAIs, err := h.resolveAIForTeam(ctx, assistanceID)  // independent call, fetches team once
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

Note on empty prompts: when `a.InitPrompt == ""` (AI has no prompt configured), `getInitPrompt()` returns `""` regardless of `activeflowID`. The `PromptSnapshot.Prompt` field will be `""`. Writing a snapshot with an empty prompt is intentional — it records that the AI had no prompt at call start.

**`ReferenceTypeTask` path (`StartTask`):** `StartTask()` calls `startAIcallByMessaging()` which contains the metadata-building logic. Task-based AIcalls therefore also capture prompt snapshots. This is intentional.

**Conversation reuse path (`startReferenceTypeConversation`):** When an AIcall is reused from an existing conversation, no new AIcall record is created. The metadata on the existing AIcall record is left unchanged (it was captured at original creation time). No new snapshot is written on reuse.

---

### D. `Create()` / `CreateByMessaging()` — accept `metadata`

Both functions gain `metadata map[string]any` as their **last parameter**, included in the SQL INSERT. The `Metadata` field on the returned `AIcall` is populated from the written value. The only callers of each function are `startAIcallByRealtime` (calls `h.Create`) and `startAIcallByMessaging` (calls `h.CreateByMessaging`) — both updated in step 7.

The metadata is built **inside** `startAIcallByRealtime()` and `startAIcallByMessaging()` (as shown in Section C) and then passed to `Create()`/`CreateByMessaging()`. The start functions themselves do **not** gain a metadata parameter — they produce metadata internally.

`StartTask()` requires **no code changes** — snapshot building is self-contained inside `startAIcallByMessaging()`, so the call from `StartTask` works correctly without modification.

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

Note: `DEFAULT (JSON_OBJECT())` requires MySQL 8.0.13+. If the deployment target is MySQL 5.7 or MySQL 8.0 < 8.0.13, use `DEFAULT '{}'` (MySQL coerces the string to JSON for a JSON column). Verify the target version before choosing the default syntax.

The two tables (`ai_ais` and `ai_aicalls`) are independent — there is no foreign-key dependency between these migrations. They are generated sequentially via two `alembic revision` commands, and Alembic automatically chains `down_revision` in the order they are created. Run Migration 1 first, then Migration 2. Before committing, verify that Migration 2's `down_revision` field matches Migration 1's revision ID to confirm correct chaining.

---

## API / OpenAPI

- `GET /ais/{id}` response: expose `current_prompt_history_id`.
- `GET /aicalls/{id}` response: expose `metadata` (customers see `prompt_snapshots` under it).
- Webhook events (`aicall.start`, `aicall.end`, etc.): expose `metadata` via `WebhookMessage`.
- No new endpoints required.

**OpenAPI spec changes (step 9):**

The OpenAPI spec lives in `bin-openapi-manager/` (generates Go types via `go generate` / oapi-codegen). Two schema additions are required:

1. In the `AI` schema object: add `current_prompt_history_id` as `type: string, format: uuid, example: "00000000-0000-0000-0000-000000000000"`.
2. In the `AIcall` schema object: add `metadata` as `type: object, additionalProperties: true` (free-form JSON map).
3. Also add `PromptSnapshot` as a named schema component with fields: `ai_id` (string/uuid), `prompt_history_id` (string/uuid), `prompt` (string), `member_id` (string/uuid).

After editing the spec YAML, run `go generate ./...` inside `bin-openapi-manager` to regenerate Go types, then update `bin-api-manager` to reference the new types. Both services need the full verification workflow run.

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
  - `StartTask` + `AssistanceTypeTeam` → `Metadata` contains one `PromptSnapshot` per team member
  - `resolveAIForTeam` → unit test for parallel fetch, concurrency-safe map construction, partial-failure behavior
- Existing tests must continue to pass (no regression).

---

## Implementation Order

1. Alembic migrations (schema first)
2. `models/ai/main.go` — add `CurrentPromptHistoryID`; `models/ai/field.go` — add `FieldCurrentPromptHistoryID`
3. `models/aicall/main.go` — add `Metadata`, `PromptSnapshot`, `MetaKeyPromptSnapshots`; `models/aicall/field.go` — add `FieldMetadata`
4. `models/aicall/webhook.go` — expose `Metadata` in `WebhookMessage` and `ConvertWebhookMessage()`
5. `pkg/dbhandler/` — `AICreate` / `AIUpdate` accept `current_prompt_history_id`; `AIcallCreate` accepts `metadata`
6. `pkg/aihandler/` — update `Create` and `Update` write sequence (use `h.db.AIUpdate` field-map for prompt-change branches)
7. `pkg/aicallhandler/` — add `resolveAIForTeam` (no `resolveAI` signature change); update `startAIcallByRealtime` / `startAIcallByMessaging` / `Create` / `CreateByMessaging`
8. RST docs update — edit `bin-api-manager/docsdev/source/` for new `current_prompt_history_id` and `metadata` fields; clean rebuild (`cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`); force-add build output (`git add -f bin-api-manager/docsdev/build/`)
9. OpenAPI spec update:
   a. Edit YAML in `bin-openapi-manager/` (add `current_prompt_history_id`, `metadata`, `PromptSnapshot` schema)
   b. `cd bin-openapi-manager && go generate ./...` — regenerate Go types
   c. Run full verification for `bin-openapi-manager`: `go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m`
   d. Update `bin-api-manager` to reference the regenerated types
   e. Run full verification for `bin-api-manager`: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
10. Tests (add/extend in `bin-ai-manager`)
11. Full verification for `bin-ai-manager`:
    ```bash
    cd bin-ai-manager
    go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
    ```
    Also run verification for any other service whose `go.mod` references were updated.
