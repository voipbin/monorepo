# Design: Store Evaluated Message IDs on AI Audits

**Date:** 2026-05-28
**Branch:** NOJIRA-aiaudit-store-message-ids
**Status:** Draft v5

---

## Problem

When an `AIAudit` evaluation runs, the audit handler loads a filtered set of messages from the DB and builds a transcript that is sent to Gemini. The resulting audit record stores the score and evaluation JSON, but does **not** record which messages were included. This makes it impossible to:

- Know exactly what Gemini evaluated (traceability / debugging)
- Expose the message set to customers via the API
- Re-run an audit against the same exact input (reproducibility)

## Goal

Store the list of message IDs that were sent to Gemini as part of the completed audit record.

---

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Storage location | JSON column on `ai_ai_audits` | Use case is read-by-audit-ID; no reverse lookup needed |
| Which IDs to store | Messages returned by `MessageList` (LIMIT 500) — exactly those rendered into the transcript | `buildTranscript` renders every returned row; when a call has >500 messages only the 500 most recent are captured (same window Gemini saw; `truncated` warning is logged) |
| Ordering | `ORDER BY tm_create DESC` — newest-first | Inherits from `MessageList`'s existing `ORDER BY tm_create desc`. Clients treat the array as the "evaluation window", not a forward-chronological transcript |
| When to populate | `completed` only; `NULL` on `progressing`, `failed`, historical | A failed audit never produced a valid evaluation. `status` is the authoritative completion signal — not `message_ids` |
| Zero-message completed audits | Store `NULL`, not `[]` | `len(messageIDs) > 0` guard intentionally collapses nil and non-nil-but-empty to NULL. Rely on `status = completed` to distinguish completion from failure |
| Column type | `JSON NULL` | Consistent with `evaluation` column (also `JSON NULL`) |
| `db:"message_ids,json"` tag | Read-path only | `copyJSON` uses `json.Unmarshal(data, &field)` which correctly handles `[]uuid.UUID` (`gofrs/uuid.UUID` implements `json.Unmarshaler`). Write path is hand-rolled in `AIAuditUpdateFinal` |
| `field.go` | No change | `message_ids` is not a filter criterion; intentionally omitted |
| Backfill | None | Historical completed audits keep `message_ids = NULL`; clients tolerate absent key |

---

## Schema Change

**Migration:** generate a new Alembic revision in `bin-dbscheme-manager`:

```bash
alembic -c alembic.ini revision -m "add_message_ids_to_ai_ai_audits"
```

Then fill in the generated file:

```python
def upgrade():
    op.execute("""
        ALTER TABLE ai_ai_audits
          ADD COLUMN message_ids JSON NULL
          AFTER evaluation
    """)

def downgrade():
    op.execute("""
        ALTER TABLE ai_ai_audits
          DROP COLUMN message_ids
    """)
```

Physical column position (`AFTER evaluation`) does not affect the Go reflection scanner.

---

## `bin-openapi-manager` Changes

Add `message_ids` to the `AIManagerAIAudit` schema in `bin-openapi-manager/openapi/openapi.yaml`, after the `prompt_history_id` block (after line 2837, before `status:`):

```yaml
        message_ids:
          type: array
          nullable: true
          items:
            type: string
            format: uuid
            x-go-type: string
          description: >
            Ordered list of message IDs (newest-first) that were evaluated by Gemini.
            Null while progressing, on failure, or for audits completed before this feature.
            Present and non-empty on successful completion for calls with messages.
          example:
            - "550e8400-e29b-41d4-a716-446655440001"
            - "550e8400-e29b-41d4-a716-446655440002"
```

Then regenerate:

```bash
cd bin-openapi-manager && go generate ./...
```

This updates `gens/models/gen.go`. Commit `openapi.yaml` and `gens/models/gen.go` together.

---

## `bin-ai-manager` Model Changes

### `models/aiaudit/main.go`

Insert after the `Evaluation` field (line 42):

```go
MessageIDs []uuid.UUID `json:"message_ids,omitempty" db:"message_ids,json"`
```

The `db:"message_ids,json"` tag is used on the **read path only**. The write path uses a hand-rolled `sql.NullString` in `AIAuditUpdateFinal`. This asymmetry is intentional.

`omitempty` means a `nil` slice produces no `message_ids` key in JSON. Client contract:

| `status` | `message_ids` key | Meaning |
|---|---|---|
| `progressing` | absent | in-flight |
| `failed` | absent | failed |
| `completed` | present, non-empty | normal successful evaluation |
| `completed` | absent | zero-message call or historical record |

### `models/aiaudit/webhook.go`

Add `MessageIDs` to `WebhookMessage` and copy it in `ConvertWebhookMessage`:

```go
// WebhookMessage
MessageIDs []uuid.UUID `json:"message_ids,omitempty"`

// ConvertWebhookMessage
MessageIDs: a.MessageIDs,
```

### `bin-api-manager/docsdev/source/` — RST doc update

Update `aiaudit_struct_*.rst` in three places:

1. **Schema block** — add `message_ids` field entry (type, nullable, description)
2. **Field description list** — add `message_ids` bullet with semantics and ordering note
3. **Example JSON block** — add `"message_ids": ["uuid1", "uuid2"]` to the completed-audit example

Clean rebuild:
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```

Commit RST source and built HTML together.

### `bin-ai-manager/docs/domain.md` — service docs sync

Update the `AIAudit` entity entry to include the new `MessageIDs` field with its semantics.

---

## `bin-ai-manager` DB Handler Changes

### Interface (`pkg/dbhandler/main.go`)

```go
AIAuditUpdateFinal(ctx context.Context, id uuid.UUID, status aiaudit.Status, overallScore *int, evaluation json.RawMessage, errStr string, messageIDs []uuid.UUID) (int64, error)
```

### Implementation (`pkg/dbhandler/aiaudit.go`)

Full updated `AIAuditUpdateFinal`:

```go
func (h *handler) AIAuditUpdateFinal(ctx context.Context, id uuid.UUID, status aiaudit.Status, overallScore *int, evaluation json.RawMessage, errStr string, messageIDs []uuid.UUID) (int64, error) {
    ts := h.utilHandler.TimeNow()

    var evalJSON sql.NullString
    if evaluation != nil {
        evalJSON = sql.NullString{String: string(evaluation), Valid: true}
    }

    var msgIDsJSON sql.NullString
    if len(messageIDs) > 0 {
        b, err := json.Marshal(messageIDs)
        if err != nil {
            return 0, fmt.Errorf("AIAuditUpdateFinal: could not marshal message_ids: %w", err)
        }
        msgIDsJSON = sql.NullString{String: string(b), Valid: true}
    }

    query := fmt.Sprintf(`
        UPDATE %s
        SET status = ?, overall_score = ?, evaluation = ?, message_ids = ?, error = ?, tm_update = ?
        WHERE id = ? AND tm_delete IS NULL AND status = 'progressing'
    `, aiauditTable)

    result, err := h.db.ExecContext(ctx, query,
        string(status),   // 1
        overallScore,     // 2
        evalJSON,         // 3
        msgIDsJSON,       // 4
        errStr,           // 5
        ts,               // 6
        id.Bytes(),       // 7 (WHERE)
    )
    if err != nil {
        return 0, fmt.Errorf("AIAuditUpdateFinal: could not execute. err: %v", err)
    }

    n, err := result.RowsAffected()
    if err != nil {
        return 0, fmt.Errorf("AIAuditUpdateFinal: could not get rows affected. err: %v", err)
    }

    return n, nil
}
```

### `AIAuditUpsert` — reset `message_ids` on re-audit

Add `message_ids = NULL` to the `ON DUPLICATE KEY UPDATE` clause alongside the existing `overall_score = NULL`, `evaluation = NULL`, `error = NULL` resets. This prevents a re-audited record that subsequently fails from retaining stale message IDs from the prior completed run:

```sql
ON DUPLICATE KEY UPDATE
    status            = IF(status = 'progressing', status, 'progressing'),
    tm_delete         = NULL,
    overall_score     = NULL,
    evaluation        = NULL,
    message_ids       = NULL,   ← add this line
    error             = NULL,
    language          = VALUES(language),
    prompt_history_id = VALUES(prompt_history_id),
    tm_update         = NULL
```

---

## `bin-ai-manager` Audit Handler Changes

### `pkg/aiaudithandler/main.go` — `runAuditJob`

The existing finalizer uses package-local vars (`finalStatus`, `finalScore`, `finalEvalJSON`, `finalErr`) that default to the failed state and are **only overwritten in the Step-7 success block**. Follow the same pattern.

**Declare alongside other `final*` vars:**
```go
var finalMsgIDs []uuid.UUID  // nil unless audit completes successfully
```

**After `MessageList` succeeds (Step 4), collect IDs locally:**
```go
msgIDs := make([]uuid.UUID, len(msgs))
for i, m := range msgs {
    msgIDs[i] = m.ID
}
```

**In the Step-7 success block only:**
```go
finalStatus = aiaudit.StatusCompleted
finalScore = &score
finalEvalJSON = rawJSON
finalMsgIDs = msgIDs  // ← only assigned here
```

**Update the deferred call:**
```go
n, dbErr := h.db.AIAuditUpdateFinal(writeCtx, recordID, finalStatus, finalScore, finalEvalJSON, finalErr, finalMsgIDs)
```

### `SweepStaleAudits` — second caller

```go
h.db.AIAuditUpdateFinal(ctx, a.ID, aiaudit.StatusFailed, nil, nil, string(aiaudit.ErrorEvaluatorUnavailable), nil)
```

---

## Mock & Test Updates

| File | Required change |
|---|---|
| `pkg/dbhandler/mock_main.go` | Regenerate — `DBHandler` interface changed |
| `pkg/aiaudithandler/mock_main.go` | Regenerate — used in `aiaudithandler` tests |
| Both via `go generate ./...` in `bin-ai-manager` | Runs all mockgen directives |
| `pkg/aiaudithandler/main_test.go` (~line 68) | Add 7th `gomock.Any()` arg to existing mock expectation |
| `pkg/aiaudithandler/main_test.go` — **new completed-path test** | Stub `MessageList` to return ≥1 real `*message.Message` values with known IDs; stub `Evaluate` to return a non-nil result. Use `DoAndReturn` on `AIAuditUpdateFinal` to capture `messageIDs` and close a done-channel. Test waits on the done-channel (with timeout) before asserting, to synchronize with the background goroutine. **Cannot reuse `TestCreate_HappyPath`** — that test stubs `MessageList` → `nil, nil` and `Evaluate` → `nil, nil, nil`, which causes the job to fail (Step-7 is never reached) |
| `pkg/aiaudithandler/main_test.go` — each failure case | Assert 7th arg to `AIAuditUpdateFinal` is `nil` |
| `pkg/dbhandler/aiaudit_test.go` — new round-trip test | Call `AIAuditUpdateFinal` with `messageIDs = []uuid.UUID{uuid1, uuid2}` (the production write path), then `AIAuditGet`, assert `result.MessageIDs` equals `[uuid1, uuid2]`. `insertTestAudit` (lines 18-47) does not include `message_ids` — no change needed there, as `JSON NULL` columns default to NULL on INSERT |

---

## Affected Services Summary

| Service | Change | Verification |
|---|---|---|
| `bin-dbscheme-manager` | New Alembic migration file | Apply and rollback migration in dev DB |
| `bin-openapi-manager` | `openapi.yaml` + `gens/models/gen.go` | `go generate ./...` then `go build ./...` |
| `bin-ai-manager` | Model, webhook, DB handler, audit handler, tests, RST docs | Full verification workflow (see below) |
| `bin-api-manager` | Downstream consumer of generated types | Full verification workflow (see below) |

---

## Verification Steps

### `bin-openapi-manager`
```bash
oapi-codegen -config configs/config_model/config.generate.yaml openapi/openapi.yaml > /dev/null
go generate ./...
go build ./...
```

### `bin-ai-manager`
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

### `bin-api-manager`
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

### Runtime verification
1. Alembic migration applies cleanly in dev DB; `downgrade()` also rolls back cleanly
2. Create an audit; confirm `message_ids` key is absent in `GET /aiaudits/:id` response while `progressing`
3. After completion, confirm `message_ids` is a non-empty JSON array matching the messages in the call
4. Simulate Gemini failure; confirm `message_ids` remains absent
5. Submit the same `aicall_id` for a second audit (re-audit); confirm the new record starts with `message_ids = NULL`
6. RST doc updated, HTML rebuilt (`rm -rf build` first), both committed together
