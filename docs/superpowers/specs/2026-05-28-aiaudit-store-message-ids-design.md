# Design: Store Evaluated Message IDs on AI Audits

**Date:** 2026-05-28
**Branch:** NOJIRA-aiaudit-store-message-ids
**Status:** Draft

---

## Problem

When an `AIAudit` evaluation runs, the audit handler loads a filtered set of messages from the DB and builds a transcript that is sent to Gemini. The resulting audit record stores the score and evaluation JSON, but does **not** record which messages were included. This makes it impossible to:

- Know exactly what Gemini evaluated (traceability / debugging)
- Expose the message set to customers via the API
- Re-run an audit against the same exact input (reproducibility)

## Goal

Store the ordered list of message IDs that were passed to Gemini as part of the completed audit record.

---

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Storage location | JSON column on `ai_ai_audits` | Use case is read-by-audit-ID; no reverse lookup needed |
| Which IDs to store | Only the messages actually sent to Gemini (post-truncation, ≤500) | Matches "evaluated" semantics exactly |
| When to populate | On `completed` only; `NULL` while `progressing` or on `failed` | A failed audit never produced a valid evaluation |
| Column type | `JSON NULL` | Consistent with `evaluation` column; nullable until populated |

---

## Schema Change

**Migration:** `bin-dbscheme-manager` Alembic migration

```sql
ALTER TABLE ai_ai_audits
  ADD COLUMN message_ids JSON NULL
  AFTER evaluation;
```

---

## Model Changes

### `bin-ai-manager/models/aiaudit/main.go`

Add one field to `AIAudit`:

```go
MessageIDs []uuid.UUID `json:"message_ids,omitempty" db:"message_ids,json"`
```

Placed after `Evaluation` to mirror column order.

### `bin-ai-manager/models/aiaudit/webhook.go`

Add `MessageIDs` to `WebhookMessage` and copy it in `ConvertWebhookMessage`:

```go
// WebhookMessage
MessageIDs []uuid.UUID `json:"message_ids,omitempty"`

// ConvertWebhookMessage
MessageIDs: a.MessageIDs,
```

---

## DB Handler Changes

### Interface (`bin-ai-manager/pkg/dbhandler/main.go`)

Update the `AIAuditUpdateFinal` signature:

```go
AIAuditUpdateFinal(ctx context.Context, id uuid.UUID, status aiaudit.Status, overallScore *int, evaluation json.RawMessage, errStr string, messageIDs []uuid.UUID) (int64, error)
```

### Implementation (`bin-ai-manager/pkg/dbhandler/aiaudit.go`)

Serialize `messageIDs` to JSON and include it in the UPDATE:

```go
var msgIDsJSON sql.NullString
if len(messageIDs) > 0 {
    b, _ := json.Marshal(messageIDs)
    msgIDsJSON = sql.NullString{String: string(b), Valid: true}
}

query := fmt.Sprintf(`
    UPDATE %s
    SET status = ?, overall_score = ?, evaluation = ?, message_ids = ?, error = ?, tm_update = ?
    WHERE id = ? AND tm_delete IS NULL AND status = 'progressing'
`, aiauditTable)
```

---

## Audit Handler Changes

### `bin-ai-manager/pkg/aiaudithandler/main.go`

In `runAuditJob`, after `MessageList` returns successfully and before `buildTranscript`:

```go
// Collect the IDs of messages that will be sent to Gemini.
msgIDs := make([]uuid.UUID, len(msgs))
for i, m := range msgs {
    msgIDs[i] = m.ID
}
```

Update the deferred `AIAuditUpdateFinal` call:
- **Success path:** pass `msgIDs`
- **Failure paths:** pass `nil` (field stays `NULL`)

The `msgIDs` variable is declared before the `defer` block and populated inside the goroutine body once messages are loaded. The `defer` closure captures it by reference so the final value is used when the deferred function executes.

---

## Data Flow (Updated)

```
runAuditJob
  │
  ├─ MessageList(aicall_id, [active_ai_id]) → msgs
  │
  ├─ Extract msgIDs from msgs          ← NEW
  │
  ├─ buildTranscript(msgs)
  │
  ├─ geminiHandler.Evaluate(transcript)
  │
  └─ defer: AIAuditUpdateFinal(
         status=completed,
         score, evalJSON,
         messageIDs=msgIDs    ← NEW (nil on failure)
     )
```

---

## Mock & Test Updates

| File | Change |
|---|---|
| `pkg/dbhandler/mock_main.go` | Regenerate with `go generate` to match new `AIAuditUpdateFinal` signature |
| `pkg/aiaudithandler/main_test.go` | Update mock expectations to include `messageIDs` arg; assert `message_ids` is populated on success and nil on failure |
| `pkg/aiaudithandler/helpers_test.go` | No change expected |

---

## Out of Scope

- OpenAPI spec update (`bin-openapi-manager`) — the `WebhookMessage` change will surface `message_ids` in JSON responses automatically; a formal OpenAPI schema update for `message_ids` is a follow-up.
- Backfilling existing audit records — historical audits will have `message_ids = NULL`.
- Storing message IDs for failed audits — intentionally excluded; a failed audit never produced a valid evaluation.

---

## Verification Steps

1. `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in `bin-ai-manager`
2. Create an audit; confirm `message_ids` is `NULL` while `progressing`
3. After completion, confirm `message_ids` is a non-empty JSON array matching the messages in the call
4. Confirm `GET /aiaudits/:id` response includes `message_ids`
5. Simulate Gemini failure; confirm `message_ids` remains `NULL`
