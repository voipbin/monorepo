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
    TMCreate *time.Time `json:"tm_create" db:"tm_create"` // pointer follows project convention (style choice for consistency, not nullability)
}
```

No `field.go` — `GetsByAIID` filters explicitly by `ai_id` and does not use the generic `filters map[Field]any` pattern. This is intentional: this entity has no user-facing filter surface beyond its parent AI.

A `WebhookMessage` / `webhook.go` is **not** needed — `AIPromptHistory` is a read-only historical record with no associated webhook events. The raw model struct is returned directly from the API. RST struct documentation must document the raw `AIPromptHistory` struct fields directly (not a `WebhookMessage` conversion layer, since none exists). This deviates from the normal pattern intentionally.

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

`tm_create` has no `DEFAULT CURRENT_TIMESTAMP` — the Go handler always sets it explicitly (matching the pattern of all other entity create methods).

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

Parameter order: `size uint64, token string` — matching all existing list methods (`AIList`, `MessageList`, `SummaryList`, etc.).

After adding these, run `go generate ./...` to regenerate `mock_main.go`.

### On AI create (`aihandler.Create` in `chatbot.go`)

```
call h.dbCreate(ctx, ...) → inserts AI row; dbCreate internally calls AIGet after insert (standard post-create pattern, unrelated to history deduplication)
if init_prompt != "":
    h.db.AIPromptHistoryCreate(ctx, &AIPromptHistory{
        ID:         h.utilHandler.NewUUID(),
        CustomerID: ai.CustomerID,
        AIID:       ai.ID,
        Prompt:     init_prompt,
        TMCreate:   h.utilHandler.TimeNow(),
    })
```

### On AI update (`aihandler.Update` in `chatbot.go`)

Since `PUT /v1/ais/{id}` always includes `init_prompt` as a positional parameter, every Update call carries an `init_prompt` value. There is no field-presence test — the real condition is:

```
if newInitPrompt != "":
    current = h.db.AIGet(ctx, id)   // MUST precede h.dbUpdate — reads pre-update value
    h.dbUpdate(ctx, ...)             // applies the SQL UPDATE + cache refresh
    if newInitPrompt != current.InitPrompt:
        h.db.AIPromptHistoryCreate(ctx, &AIPromptHistory{...})
else:
    h.dbUpdate(ctx, ...)             // newInitPrompt is empty; skip pre-fetch and history insert
```

**Temporal constraint:** The pre-fetch `AIGet` call **must** happen before `h.dbUpdate` to read the pre-update value. After `dbUpdate`, the cache holds the new value.

**Deduplication:** Skip insert if `newInitPrompt == current.InitPrompt` — no identical consecutive versions stored.

**Empty prompt:** Skip insert (and pre-fetch) if `newInitPrompt == ""`.

---

## New Handler Package

### `AIPromptHistoryHandler` interface

`pkg/aiprompthistoryhandler/main.go`:

```go
//go:generate mockgen -destination mock_main.go -package aiprompthistoryhandler . AIPromptHistoryHandler

type AIPromptHistoryHandler interface {
    // List returns prompt history entries for the given AI, newest first.
    // No customer_id parameter — follows the same authorization model as aihandler.Get
    // (ID-based scoping; gateway-level isolation assumed).
    List(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)

    // Get returns a single prompt history entry by its ID.
    // Returns an error if history.AIID != aiID (enforced at handler layer, not DB layer).
    Get(ctx context.Context, aiID uuid.UUID, historyID uuid.UUID) (*aiprompthistory.AIPromptHistory, error)
}
```

**Authorization model:** follows the same pattern as `aihandler.Get` — no explicit `customer_id` parameter. Single-resource gets are scoped by the resource ID only. The `ai_id` path parameter provides implicit scoping: `Get` verifies `history.AIID == aiID`; if not, it returns a not-found error. This cross-check is enforced in the handler layer — `AIPromptHistoryGet` fetches by `historyID` only (no `ai_id` in the DB method signature). The `customer_id` field is stored on history rows for potential future use but is not actively used for authorization in v1.

### Constructor

```go
func New(db dbhandler.DBHandler, util utilhandler.UtilHandler) AIPromptHistoryHandler
```

Follows the convention of all other handler packages in this service.

### Wiring into `listenhandler`

`pkg/listenhandler/main.go` — `listenHandler` struct gains:

```go
aiprompthistoryHandler aiprompthistoryhandler.AIPromptHistoryHandler
```

`NewListenHandler(...)` gains `aiprompthistoryHandler aiprompthistoryhandler.AIPromptHistoryHandler` as a parameter.

`cmd/ai-manager/main.go` — constructs `aiprompthistoryhandler.New(db, util)` and passes it to `NewListenHandler`. No new config flags; `docs/operations.md` does not need updating.

### New route patterns

Following the existing convention (no `^` anchor; list route matches `\?`, single-entry route matches `$`):

```go
// matches GET /v1/ais/{uuid}/prompt_histories?...
regV1AIsIDPromptHistoriesGet = regexp.MustCompile(`/v1/ais/` + regUUID + `/prompt_histories\?`)

// matches GET /v1/ais/{uuid}/prompt_histories/{uuid}
regV1AIsIDPromptHistoriesIDGet = regexp.MustCompile(`/v1/ais/` + regUUID + `/prompt_histories/` + regUUID + `$`)
```

### URI parsing

**List route** (uses `url.Parse` — same as `processV1AIsGet`, because the URI contains a query string):

```go
u, err := url.Parse(m.URI)
// ai_id is in u.Path: strings.Split(u.Path, "/")[3]
// size, token from u.Query()
```

**Get-one route** (uses `strings.Split` — same as `processV1AIsIDGet`, clean path):

```go
uriItems := strings.Split(m.URI, "/")
// path: "" / "v1" / "ais" / <ai_id> / "prompt_histories" / <history_id>
// index:  0     1     2       3            4                    5
if len(uriItems) < 6 { ... }
aiID      := uuid.FromStringOrNil(uriItems[3])
historyID := uuid.FromStringOrNil(uriItems[5])
```

---

## DB Layer

### New file: `pkg/dbhandler/aiprompthistory.go`

```go
// AIPromptHistoryCreate inserts a new AIPromptHistory row. TMCreate must be set by the caller.
func (h *handler) AIPromptHistoryCreate(ctx context.Context, p *aiprompthistory.AIPromptHistory) error

// AIPromptHistoryGet returns a single entry by ID (no ai_id parameter — ai_id cross-check
// is enforced in the handler layer, not the DB layer, consistent with aihandler.Get).
func (h *handler) AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error)

// AIPromptHistoryGetsByAIID returns entries for the given AI, newest first.
// Parameter order: size uint64, token string — matches all other list methods.
func (h *handler) AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)
```

No cache-aside for history rows — rows are immutable but accessed infrequently. Extending the `cachehandler` interface for this is out of scope for v1.

---

## API Endpoints

Both endpoints are read-only. Authorization follows the same model as `aihandler.Get`: ID-based scoping, no explicit customer_id parameter. Gateway-level isolation is assumed for multi-tenant safety.

### List prompt history

```
GET /v1/ais/{ai_id}/prompt_histories?size=N&token=T
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
| No history rows for `ai_id` | empty array `[]` (list returns empty, not 404) |
| `history_id` not found | `404 Not Found` |
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
- List: returns entries for valid `ai_id`
- List: returns empty slice when no history rows exist for `ai_id`
- Get: returns single entry when `history.AIID == aiID`
- Get: returns not-found error when `history.AIID ≠ aiID`
- Get: returns not-found error when `historyID` does not exist

### `pkg/aihandler/chatbot_test.go` (extend existing)
- Create AI with non-empty `init_prompt` → `AIPromptHistoryCreate` called after `dbCreate`
- Create AI with empty `init_prompt` → `AIPromptHistoryCreate` NOT called
- Update AI with non-empty `init_prompt` that differs from current → `AIGet` pre-fetch (before `dbUpdate`), then `dbUpdate`, then `AIPromptHistoryCreate`
- Update AI with non-empty `init_prompt` same as current → `AIGet` pre-fetch, then `dbUpdate`; `AIPromptHistoryCreate` NOT called
- Update AI with empty `init_prompt` → `dbUpdate` only; no `AIGet` pre-fetch, no `AIPromptHistoryCreate`

### `pkg/listenhandler/` (extend existing, e.g. `v1_ais_test.go` or new `v1_ai_prompt_histories_test.go`)
- List: happy path returns `200` with JSON array
- List: invalid `ai_id` UUID in path returns `400`
- Get: happy path returns `200` with single entry
- Get: invalid `ai_id` or `history_id` UUID in path returns `400`
- Get: not-found from handler returns appropriate error response

---

## Restore Flow (no new endpoint)

Restore is a client-side operation:

1. `GET /v1/ais/{ai_id}/prompt_histories?size=50` — find desired version
2. Copy the `prompt` field value
3. `PUT /v1/ais/{ai_id}` with the full AI update body including `"init_prompt": "<copied value>"` — triggers a new history entry

Note: the update verb is `PUT` (not `PATCH`) — this service uses `PUT` for AI updates.

---

## Service Docs to Update

Per monorepo doc-sync rules, the following must be updated in the same commit as the implementation:

- `bin-ai-manager/docs/architecture.md` — add two new route entries for the history endpoints
- `bin-ai-manager/docs/domain.md` — add `AIPromptHistory` entity description
- `bin-ai-manager/docs/operations.md` — no changes needed (no new config flags)
- `bin-api-manager/docsdev/source/` — add RST documentation for the new endpoints and `AIPromptHistory` struct fields (document raw model struct directly — no `WebhookMessage` conversion layer exists for this entity); rebuild HTML (`rm -rf build && python3 -m sphinx -M html source build`) and force-add the build output

---

## Out of Scope

- User-level attribution (who changed the prompt) — not needed for v1
- Dedicated restore endpoint — client-side copy is sufficient
- Automatic pruning / retention policy — history is kept indefinitely
- Diff view between versions — presentation concern, handled by clients
- Webhook events for prompt history changes — no operational need
- Cache-aside for history DB reads — infrequent access does not justify extending cachehandler scope
- Customer-id parameter in handler interface — not used in v1 (consistent with existing aihandler.Get pattern)
