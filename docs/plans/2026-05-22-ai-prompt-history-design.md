# AI Prompt History — Design

**Date:** 2026-05-22
**Branch:** NOJIRA-Add-ai-prompt-history
**Status:** Design

---

## Problem

The `AI` entity stores a single `init_prompt` field. When the prompt is updated, the previous value is lost. Operators cannot:

- Determine what system prompt was active during a past AIcall (debugging)
- Recover a previous prompt after a bad update (prompt engineering iteration)

---

## Solution

Track every value that `init_prompt` takes over time in an immutable history table (`ai_ai_prompt_histories`). Expose a read-only API to list and retrieve historical prompt versions. Restore is performed by the client reading an old version and calling the normal update endpoint.

---

## Data Model

### New table: `ai_ai_prompt_histories`

```
Column      Type        Notes
──────────────────────────────────────────────────────
id          UUID        PK — unique identity of the history entry
customer_id UUID        FK — copied from parent AI at insert time
ai_id       UUID        FK → ai_ais.id
prompt      TEXT        the prompt content at this point in time
tm_create   TIMESTAMP   when this version was created (version marker)
```

No `tm_update` or `tm_delete` — rows are immutable. History is append-only.

### New Go model: `models/aiprompthistory/`

```go
package aiprompthistory

import (
    "time"

    "github.com/gofrs/uuid"

    "monorepo/bin-common-handler/models/identity"
)

type AIPromptHistory struct {
    identity.Identity               // ID + CustomerID — mandatory per monorepo convention

    AIID     uuid.UUID  `json:"ai_id"     db:"ai_id,uuid"`
    Prompt   string     `json:"prompt"    db:"prompt"`
    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}
```

### Naming conventions applied

- **`identity.Identity` always embedded** unless explicitly told not to.
- **Table name follows `{service_prefix}_{entity_plural}`** — `ai_` prefix for `bin-ai-manager`, entity `ai_prompt_histories`.

---

## Database Migration

New Alembic migration in `bin-dbscheme-manager`:

```sql
-- upgrade
CREATE TABLE ai_ai_prompt_histories (
    id          BINARY(16)   NOT NULL,
    customer_id BINARY(16)   NOT NULL,
    ai_id       BINARY(16)   NOT NULL,
    prompt      LONGTEXT     NOT NULL DEFAULT '',
    tm_create   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_ai_ai_prompt_histories_ai_id (ai_id),
    KEY idx_ai_ai_prompt_histories_customer_id (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- downgrade
DROP TABLE IF EXISTS ai_ai_prompt_histories;
```

---

## Write Path

History rows are written inside the existing `aihandler` — no new write-side handler needed.

### On AI create (`aihandler.AICreate`)

```
INSERT into ai_ais
if init_prompt != "":
    INSERT into ai_ai_prompt_histories (id=newUUID, customer_id=ai.CustomerID, ai_id=ai.ID, prompt=init_prompt, tm_create=now)
```

### On AI update (`aihandler.AIUpdate`)

```
UPDATE ai_ais SET ...
if fields contains init_prompt AND new value != current value:
    INSERT into ai_ai_prompt_histories (id=newUUID, customer_id=ai.CustomerID, ai_id=ai.ID, prompt=new_init_prompt, tm_create=now)
```

**Deduplication:** Skip insert if new `init_prompt` equals the current value — no identical consecutive versions stored.

**Empty prompt:** Skip insert if `init_prompt` is `""` — empty prompts are not meaningful history entries.

---

## Read Path

### New handler package: `pkg/aiprompthistoryhandler/`

Owns the read path for history entries. Thin layer over the DB handler.

### New DB file: `pkg/dbhandler/aiprompthistory.go`

```go
Create(ctx, AIPromptHistory) error
Get(ctx, id uuid.UUID) (AIPromptHistory, error)
GetsByAIID(ctx, aiID uuid.UUID, pageToken string, pageSize uint32) ([]AIPromptHistory, error)
```

Results ordered `tm_create DESC` (newest first).

---

## API Endpoints

Both endpoints are read-only. Authorization: caller's `customer_id` must match the parent AI's `customer_id`.

### List prompt history

```
GET /v1/ais/{ai_id}/prompt_histories
```

Query params: `page_token`, `page_size` (standard pagination)

Response:
```json
{
  "result": [
    {
      "id": "...",
      "customer_id": "...",
      "ai_id": "...",
      "prompt": "You are a helpful assistant.",
      "tm_create": "2026-05-22T10:00:00Z"
    }
  ],
  "next_page_token": "..."
}
```

### Get one prompt history entry

```
GET /v1/ais/{ai_id}/prompt_histories/{history_id}
```

Response: single `AIPromptHistory` object.

---

## Error Handling

| Condition | Response |
|-----------|----------|
| `ai_id` not found | `404 Not Found` |
| caller `customer_id` ≠ AI `customer_id` | `404 Not Found` (no information leakage) |
| `history_id` not found | `404 Not Found` |
| `history.AIID ≠ ai_id` in URL | `404 Not Found` |
| `init_prompt` empty on create/update | skip history insert silently |
| `init_prompt` unchanged on update | skip history insert silently |

---

## Testing

### `pkg/dbhandler/aiprompthistory_test.go`
- Create: inserts a row and retrieves it by ID
- Get: returns correct entry, returns error for unknown ID
- GetsByAIID: returns entries in `tm_create DESC` order, pagination works

### `pkg/aiprompthistoryhandler/` (unit tests, gomock, table-driven)
- List: returns entries for valid AI + matching customer
- List: returns `404` when AI not found
- List: returns `404` when customer mismatch
- Get: returns single entry for valid IDs
- Get: returns `404` when `history.AIID ≠ ai_id`

### `pkg/aihandler/db_test.go` (extend existing)
- Create AI with non-empty `init_prompt` → history row inserted
- Create AI with empty `init_prompt` → no history row
- Update AI changing `init_prompt` → new history row inserted
- Update AI with same `init_prompt` → no history row inserted
- Update AI without touching `init_prompt` → no history row inserted

---

## Restore Flow (no new endpoint)

Restore is a client-side operation:

1. `GET /v1/ais/{ai_id}/prompt_histories` — find desired version
2. Copy the `prompt` field value
3. `PATCH /v1/ais/{ai_id}` with `{"init_prompt": "<copied value>"}` — triggers a new history entry

---

## Out of Scope

- User-level attribution (who changed the prompt) — not needed for v1
- Dedicated restore endpoint — client-side copy is sufficient
- Automatic pruning / retention policy — history is kept indefinitely
- Diff view between versions — presentation concern, handled by clients
