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
tm_create   TIMESTAMP   NOT NULL; set explicitly by handler (not DB default)
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
    TMCreate *time.Time `json:"tm_create" db:"tm_create"` // pointer follows project convention (matches Summary.TMCreate)
}
```

No `field.go` — `GetsByAIID` filters explicitly by `ai_id` and does not use the generic `filters map[Field]any` pattern. This is an intentional deviation because this entity has no user-facing filter surface beyond its parent AI.

A `WebhookMessage` / `webhook.go` is **not** needed — `AIPromptHistory` is a read-only historical record with no associated webhook events. The raw model struct is returned directly from the API. This is intentional: webhook types exist only for entities that emit events.

**Empty-prompt behaviour:** An AI created with `init_prompt = ""` will have no history entry. If such an AI is later updated to a non-empty prompt, the history will start from that first non-empty value. This gap is accepted by design — an empty prompt produces no meaningful history.

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

`tm_create` has no `DEFAULT CURRENT_TIMESTAMP` — the Go handler always sets it explicitly (matching the pattern of all other entity create methods, e.g., `Summary.TMCreate = h.utilHandler.TimeNow()`).

---

## Write Path

History rows are written from the **public methods in `aihandler/chatbot.go`** (`Create` and `Update`), after input validation and after the DB write completes. This placement is consistent with where webhook publish (`notifyHandler.PublishWebhookEvent`) is called — at the coordination level, not inside the private `db.go` methods.

### DBHandler interface change

`pkg/dbhandler/main.go` — add to `DBHandler` interface:

```go
AIPromptHistoryCreate(ctx context.Context, h *aiprompthistory.AIPromptHistory) error
AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error)
AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)
```

Note: `size uint64, token string` — matching the parameter order of all existing list methods (`AIList`, `MessageList`, `SummaryList`, etc.).

After adding these, run `go generate ./...` to regenerate `mock_main.go`.

### On AI create (`aihandler.Create` in `chatbot.go`)

```
call h.dbCreate(ctx, ...) → inserts AI row; existing dbCreate already calls AIGet post-create internally
if init_prompt != "":
    h.db.AIPromptHistoryCreate(ctx, &AIPromptHistory{
        ID:         h.utilHandler.NewUUID(),
        CustomerID: ai.CustomerID,
        AIID:       ai.ID,
        Prompt:     init_prompt,
        TMCreate:   h.utilHandler.TimeNow(),
    })
```

Note: the existing `dbCreate` calls `AIGet` internally after insert (to retrieve the freshly created row). This is unrelated to history — it is not a "pre-fetch for deduplication." The history insert happens at the `Create` level (in `chatbot.go`) after `dbCreate` returns.

### On AI update (`aihandler.Update` in `chatbot.go`)

Deduplication requires knowing the current `init_prompt` before the update is applied. The pre-fetch **must** happen before `h.dbUpdate(ctx, ...)` is called, so that it reads the pre-update value:

```
if fields contains init_prompt AND new_init_prompt != "":
    current = h.db.AIGet(ctx, id)   // MUST precede h.dbUpdate — reads pre-update value
    h.dbUpdate(ctx, ...)             // applies the SQL UPDATE + cache refresh
    if new_init_prompt != current.InitPrompt:
        h.db.AIPromptHistoryCreate(ctx, &AIPromptHistory{...})
else:
    h.dbUpdate(ctx, ...)             // no history work needed; skip pre-fetch entirely
```

If `init_prompt` is not present in `fields`, skip the pre-fetch and the history insert entirely — no unnecessary DB read.

---

## New Handler Package

### `AIPromptHistoryHandler` interface

`pkg/aiprompthistoryhandler/main.go`:

```go
//go:generate mockgen -destination mock_main.go -package aiprompthistoryhandler . AIPromptHistoryHandler

type AIPromptHistoryHandler interface {
    List(ctx context.Context, callerCustomerID uuid.UUID, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)
    Get(ctx context.Context, callerCustomerID uuid.UUID, aiID uuid.UUID, historyID uuid.UUID) (*aiprompthistory.AIPromptHistory, error)
}
```

### Constructor

```go
func New(db dbhandler.DBHandler, util utilhandler.UtilHandler) AIPromptHistoryHandler
```

`utilhandler.UtilHandler` is included following the convention of all other handler packages in this service.

The handler does **not** need a reference to `aihandler` — authorization is enforced directly using the `customer_id` field denormalized onto each history row.

### Wiring into `listenhandler`

`pkg/listenhandler/main.go` — `listenHandler` struct gains:

```go
aiprompthistoryHandler aiprompthistoryhandler.AIPromptHistoryHandler
```

`NewListenHandler(...)` gains `aiprompthistoryHandler aiprompthistoryhandler.AIPromptHistoryHandler` as a parameter.

`cmd/ai-manager/main.go` — constructs `aiprompthistoryhandler.New(db, util)` and passes it to `NewListenHandler`. No new config flags are introduced; `docs/operations.md` does not need updating.

### New route patterns

```go
// matches GET /v1/ais/{uuid}/prompt_histories
regV1AIsIDPromptHistoriesList = regexp.MustCompile(`^/v1/ais/[0-9a-f-]+/prompt_histories$`)

// matches GET /v1/ais/{uuid}/prompt_histories/{uuid}
regV1AIsIDPromptHistoriesID = regexp.MustCompile(`^/v1/ais/[0-9a-f-]+/prompt_histories/[0-9a-f-]+$`)
```

### URI parsing (path split indices)

Path: `/v1/ais/{ai_id}/prompt_histories[/{history_id}]`

```
index: 0    1    2      3        4                   5
split: ""  "v1" "ais"  <ai_id>  "prompt_histories"  <history_id>
```

For both routes, minimum check: `len(uriItems) >= 5`. For the single-entry route: `len(uriItems) >= 6`. `ai_id` is at index `3`; `history_id` is at index `5`.

---

## DB Layer

### New file: `pkg/dbhandler/aiprompthistory.go`

```go
// AIPromptHistoryCreate inserts a new AIPromptHistory row. TMCreate must be set by the caller.
func (h *handler) AIPromptHistoryCreate(ctx context.Context, p *aiprompthistory.AIPromptHistory) error

// AIPromptHistoryGet returns a single entry by ID. No cache — history rows are accessed
// infrequently enough that adding them to the cache would expand cachehandler scope unnecessarily.
func (h *handler) AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error)

// AIPromptHistoryGetsByAIID returns entries for the given AI, newest first.
// Parameter order: size uint64, token string — matches all other list methods.
// No cache — same rationale as AIPromptHistoryGet.
func (h *handler) AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)
```

Results ordered `tm_create DESC` (newest first).

No cache-aside for history rows — rows are immutable but accessed infrequently. Extending the `cachehandler` interface for this is out of scope for v1.

---

## API Endpoints

Both endpoints are read-only.

### Authorization

Authorization uses the `customer_id` field denormalized onto each history row — no separate AI lookup is required:

- **List:** `GetsByAIID` filters by `ai_id`; handler checks `history.CustomerID == callerCustomerID` for each returned entry (or adds `customer_id` to the SQL `WHERE` clause). If no rows match, return `404`.
- **Get:** after fetching the row, check `history.CustomerID == callerCustomerID` and `history.AIID == aiID`. Either mismatch → `404`.

This works correctly even when the parent AI is soft-deleted, since history rows carry their own `customer_id`.

### List prompt history

```
GET /v1/ais/{ai_id}/prompt_histories
```

Query params: `size` (page size), `token` (page token) — matching the convention of all other list endpoints.

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
| No history rows match `ai_id` + `customer_id` (list) | `404 Not Found` |
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
- GetsByAIID: returns entries in `tm_create DESC` order; pagination (`size`/`token`) works correctly
- GetsByAIID: returns empty slice when no entries exist for given `ai_id`

### `pkg/aiprompthistoryhandler/` (unit tests, gomock, table-driven)
- List: returns entries for valid `ai_id` + matching `customer_id`
- List: returns `404` when no history rows match `ai_id` + `customer_id`
- Get: returns single entry for valid IDs
- Get: returns `404` when `history.CustomerID ≠ caller.CustomerID`
- Get: returns `404` when `history.AIID ≠ ai_id`

### `pkg/aihandler/chatbot_test.go` / `db_test.go` (extend existing)
- Create AI with non-empty `init_prompt` → `dbCreate` called (which internally calls `AIGet` post-insert); then `AIPromptHistoryCreate` called at the `Create` level
- Create AI with empty `init_prompt` → `AIPromptHistoryCreate` NOT called
- Update AI with `init_prompt` in fields, value differs from current → mock expects: `AIGet` pre-fetch (before `dbUpdate`), then `dbUpdate`, then `AIPromptHistoryCreate`
- Update AI with `init_prompt` in fields, value same as current → mock expects: `AIGet` pre-fetch, then `dbUpdate`; `AIPromptHistoryCreate` NOT called
- Update AI without `init_prompt` in fields → mock expects: `dbUpdate` only; no `AIGet` pre-fetch, no `AIPromptHistoryCreate`

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
- `bin-ai-manager/docs/operations.md` — no changes needed (no new config flags)

---

## Out of Scope

- User-level attribution (who changed the prompt) — not needed for v1
- Dedicated restore endpoint — client-side copy is sufficient
- Automatic pruning / retention policy — history is kept indefinitely
- Diff view between versions — presentation concern, handled by clients
- Webhook events for prompt history changes — no operational need
- Cache-aside for history DB reads — infrequent access does not justify extending cachehandler scope
