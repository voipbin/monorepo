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
tm_create   TIMESTAMP   when this version was created (version marker); set by handler, not DB default
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

No `field.go` is needed — `GetsByAIID` filters explicitly by `ai_id` and does not use the generic `filters map[Field]any` pattern. This is an intentional deviation from the model package convention because this entity has no user-facing filter surface beyond its parent AI.

A `WebhookMessage` / `webhook.go` is **not** needed — `AIPromptHistory` is a read-only historical record with no associated webhook events. The raw model struct is returned directly from the API. This is an intentional deviation: webhook types exist only for entities that emit events.

### Naming conventions applied

- **`identity.Identity` always embedded** unless explicitly told not to.
- **Table name follows `{service_prefix}_{entity_plural}`** — `ai_` prefix for `bin-ai-manager`, entity `ai_prompt_histories` → full table name `ai_ai_prompt_histories`.

---

## Database Migration

New Alembic migration in `bin-dbscheme-manager` (generated via `alembic revision`, never hand-crafted):

```sql
-- upgrade
CREATE TABLE ai_ai_prompt_histories (
    id          BINARY(16)   NOT NULL,
    customer_id BINARY(16)   NOT NULL,
    ai_id       BINARY(16)   NOT NULL,
    prompt      LONGTEXT     NOT NULL DEFAULT '',
    tm_create   DATETIME(6)  NOT NULL,
    PRIMARY KEY (id),
    KEY idx_ai_ai_prompt_histories_ai_id (ai_id),
    KEY idx_ai_ai_prompt_histories_customer_id (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- downgrade
DROP TABLE IF EXISTS ai_ai_prompt_histories;
```

Note: `tm_create` has no `DEFAULT CURRENT_TIMESTAMP` because the Go handler always sets it explicitly (matching the pattern of all other entity create methods).

---

## Write Path

History rows are written inside the existing `aihandler` — no new write-side handler needed. However, `aihandler` requires access to `AIPromptHistoryCreate` via the `DBHandler` interface.

### DBHandler interface change

`pkg/dbhandler/main.go` — add to `DBHandler` interface:

```go
AIPromptHistoryCreate(ctx context.Context, h *aiprompthistory.AIPromptHistory) error
AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error)
AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, token string, size uint64) ([]*aiprompthistory.AIPromptHistory, error)
```

After adding these, run `go generate ./...` to regenerate `mock_main.go`.

### On AI create (`aihandler.AICreate`)

```
INSERT into ai_ais (new AI row)
if init_prompt != "":
    h.db.AIPromptHistoryCreate(ctx, &AIPromptHistory{
        ID:         newUUID,
        CustomerID: ai.CustomerID,
        AIID:       ai.ID,
        Prompt:     init_prompt,
        TMCreate:   h.utilHandler.TimeNow(),
    })
```

### On AI update (`aihandler.AIUpdate`)

The update flow must read the current AI state **before** writing the update, in order to detect whether `init_prompt` has changed:

```
current = h.db.AIGet(ctx, id)          // pre-fetch current state
UPDATE ai_ais SET ...
if fields contains init_prompt:
    if new_init_prompt == "" → skip
    if new_init_prompt == current.InitPrompt → skip (deduplication)
    else:
        h.db.AIPromptHistoryCreate(ctx, &AIPromptHistory{...})
```

**Pre-fetch rationale:** `aihandler.Update` currently does not read the current AI before writing. Adding `h.db.AIGet(ctx, id)` is required solely for the deduplication check. The mock expectation for this call must be included in all relevant `db_test.go` test cases.

**Deduplication:** Skip insert if new `init_prompt` equals the current value — no identical consecutive versions stored.

**Empty prompt:** Skip insert if `init_prompt` is `""` — empty prompts are not meaningful history entries.

---

## New Handler Package

### `pkg/aiprompthistoryhandler/`

Owns the read path for history entries. Thin layer over the DB handler.

Constructor:

```go
func New(db dbhandler.DBHandler) AIPromptHistoryHandler
```

The handler does **not** need a reference to `aihandler` — authorization is enforced directly using the `customer_id` field denormalized onto each history row (see Authorization section below).

### Wiring into `listenhandler`

`pkg/listenhandler/main.go` — `listenHandler` struct gains:

```go
aiprompthistoryHandler aiprompthistoryhandler.AIPromptHistoryHandler
```

`NewListenHandler(...)` gains a corresponding parameter and wires the new handler to the two new route regexes.

`cmd/ai-manager/main.go` — constructs `aiprompthistoryhandler.New(db)` and passes it to `NewListenHandler`.

---

## DB Layer

### New file: `pkg/dbhandler/aiprompthistory.go`

```go
// Create inserts a new AIPromptHistory row. TMCreate must be set by the caller.
func (h *handler) AIPromptHistoryCreate(ctx context.Context, p *aiprompthistory.AIPromptHistory) error

// AIPromptHistoryGet returns a single entry by ID.
func (h *handler) AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error)

// AIPromptHistoryGetsByAIID returns entries for the given AI, newest first.
// Follows the same token/size pagination convention as all other GetsByX methods.
func (h *handler) AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, token string, size uint64) ([]*aiprompthistory.AIPromptHistory, error)
```

Results ordered `tm_create DESC` (newest first).

---

## API Endpoints

Both endpoints are read-only.

### Authorization

Authorization uses the `customer_id` field denormalized onto each history row — no separate AI lookup is required:

- **List:** filter SQL by `customer_id = caller.CustomerID AND ai_id = ai_id`. If no rows match, return `404`.
- **Get:** after fetching the row, check `history.CustomerID == caller.CustomerID` and `history.AIID == aiID`. Either mismatch → `404`.

This works correctly even when the parent AI is soft-deleted, since history rows carry their own `customer_id`.

### List prompt history

```
GET /v1/ais/{ai_id}/prompt_histories
```

Query params: `token` (page token), `size` (page size) — matching the convention of all other list endpoints.

Response: bare JSON array (no envelope wrapper — consistent with all other list endpoints in this service):

```json
[
  {
    "id": "...",
    "customer_id": "...",
    "ai_id": "...",
    "prompt": "You are a helpful assistant.",
    "tm_create": "2026-05-22T10:00:00Z"
  }
]
```

### Get one prompt history entry

```
GET /v1/ais/{ai_id}/prompt_histories/{history_id}
```

Response: single `AIPromptHistory` JSON object.

---

## Error Handling

| Condition | Response |
|-----------|----------|
| `ai_id` + `customer_id` combo returns no rows (list) | `404 Not Found` |
| `history_id` not found | `404 Not Found` |
| `history.CustomerID ≠ caller.CustomerID` | `404 Not Found` (no information leakage) |
| `history.AIID ≠ ai_id` in URL | `404 Not Found` |
| `init_prompt` empty on create/update | skip history insert silently |
| `init_prompt` unchanged on update | skip history insert silently |

---

## Testing

### `pkg/dbhandler/aiprompthistory_test.go`
- Create: inserts a row and retrieves it by ID
- Get: returns correct entry; returns error for unknown ID
- GetsByAIID: returns entries in `tm_create DESC` order; pagination (token/size) works correctly
- GetsByAIID: returns empty slice when no entries exist for given `ai_id`

### `pkg/aiprompthistoryhandler/` (unit tests, gomock, table-driven)
- List: returns entries for valid `ai_id` + matching `customer_id`
- List: returns `404` when no history rows match `ai_id` + `customer_id`
- Get: returns single entry for valid IDs
- Get: returns `404` when `history.CustomerID ≠ caller.CustomerID`
- Get: returns `404` when `history.AIID ≠ ai_id`

### `pkg/aihandler/db_test.go` (extend existing tests)
- Create AI with non-empty `init_prompt` → history row inserted (mock expects `AIGet` pre-fetch + `AIPromptHistoryCreate`)
- Create AI with empty `init_prompt` → no history row inserted
- Update AI changing `init_prompt` → `AIGet` pre-fetch called, new history row inserted
- Update AI with same `init_prompt` as current → `AIGet` pre-fetch called, no history row inserted
- Update AI without `init_prompt` in fields → no `AIGet` pre-fetch, no history row inserted

---

## Restore Flow (no new endpoint)

Restore is a client-side operation:

1. `GET /v1/ais/{ai_id}/prompt_histories` — find desired version
2. Copy the `prompt` field value
3. `PUT /v1/ais/{ai_id}` with the full AI update body including `"init_prompt": "<copied value>"` — triggers a new history entry

Note: the update verb is `PUT` (not `PATCH`) — this service uses `PUT` for AI updates.

---

## Service Docs to Update

Per monorepo doc-sync rules, the following must be updated in the same commit as the implementation:

- `bin-ai-manager/docs/architecture.md` — add two new route entries for the history endpoints
- `bin-ai-manager/docs/domain.md` — add `AIPromptHistory` entity description
- `bin-api-manager/docsdev/source/` — add RST documentation for the new endpoints and `AIPromptHistory` struct; rebuild HTML (`rm -rf build && python3 -m sphinx -M html source build`) and force-add the build output

---

## Out of Scope

- User-level attribution (who changed the prompt) — not needed for v1
- Dedicated restore endpoint — client-side copy is sufficient
- Automatic pruning / retention policy — history is kept indefinitely
- Diff view between versions — presentation concern, handled by clients
- Webhook events for prompt history changes — no operational need
