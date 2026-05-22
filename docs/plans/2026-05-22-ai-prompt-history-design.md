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

## Scope

This design covers the full exposure chain across four services:

1. **`bin-ai-manager`** — data model, DB layer, write path, internal handler, listen routes
2. **`bin-common-handler`** — requesthandler methods for inter-service RPC
3. **`bin-openapi-manager`** — OpenAPI YAML spec for the two new routes
4. **`bin-api-manager`** — HTTP handlers and service-layer methods; must follow the Two-Level Handler Pattern

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
tm_create   TIMESTAMP   NOT NULL; set by dbhandler (matching all other entity create methods)
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
    TMCreate *time.Time `json:"tm_create" db:"tm_create"` // pointer follows project convention (style choice, not nullability)
}
```

No `field.go` — `GetsByAIID` filters explicitly by `ai_id` and does not use the generic `filters map[Field]any` pattern. This is intentional: this entity has no user-facing filter surface beyond its parent AI.

No `webhook.go` — `AIPromptHistory` is a read-only historical record with no associated webhook events. All fields in `AIPromptHistory` are appropriate for external consumers, so `bin-api-manager` will serialize the raw struct directly. This requires an explicit note in the `bin-api-manager` servicehandler to document the justified deviation from the `ConvertWebhookMessage()` convention.

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

`tm_create` has no `DEFAULT CURRENT_TIMESTAMP` — the dbhandler sets it explicitly via `h.utilHandler.TimeNow()` (see DB Layer section).

---

## Write Path (`bin-ai-manager`)

History rows are written from the **public methods in `aihandler/chatbot.go`** (`Create` and `Update`), after input validation and after the DB write completes. This placement is consistent with where webhook publish (`notifyHandler.PublishWebhookEvent`) is called — at the coordination level, not inside the private `db.go` methods.

**Error handling:** `AIPromptHistoryCreate` is a best-effort side effect — if it fails, the failure is logged and the overall operation succeeds. The AI update is not rolled back. This matches the webhook publish pattern.

### DBHandler interface change (`pkg/dbhandler/main.go`)

Add to `DBHandler` interface:

```go
AIPromptHistoryCreate(ctx context.Context, h *aiprompthistory.AIPromptHistory) error
AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error)
AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)
```

Parameter order: `size uint64, token string` — matching all existing list methods (`AIList`, `MessageList`, `SummaryList`, etc.).

After adding these, run `go generate ./...` to regenerate `mock_main.go`.

### On AI create (`aihandler.Create` in `chatbot.go`)

```
call h.dbCreate(ctx, ...) → inserts AI row; dbCreate internally calls AIGet after insert (standard post-create get, unrelated to history)
if init_prompt != "":
    h.db.AIPromptHistoryCreate(ctx, &AIPromptHistory{
        ID:         h.utilHandler.UUIDCreate(),   // UUIDCreate(), not NewUUID()
        CustomerID: ai.CustomerID,
        AIID:       ai.ID,
        Prompt:     init_prompt,
        // TMCreate is NOT set here — set by dbhandler (see DB Layer)
    })
    // log error if AIPromptHistoryCreate fails; do not fail the Create operation
```

### On AI update (`aihandler.Update` in `chatbot.go`)

Since `PUT /v1/ais/{id}` always includes `init_prompt` as a positional parameter, every Update call carries an `init_prompt` value. There is no field-presence test — the real condition is:

```
if newInitPrompt != "":
    current = h.db.AIGet(ctx, id)   // MUST precede h.dbUpdate — reads pre-update value
    h.dbUpdate(ctx, ...)             // applies the SQL UPDATE + cache refresh
    if newInitPrompt != current.InitPrompt:
        h.db.AIPromptHistoryCreate(ctx, &AIPromptHistory{
            ID:         h.utilHandler.UUIDCreate(),
            CustomerID: current.CustomerID,
            AIID:       current.ID,
            Prompt:     newInitPrompt,
        })
        // log error if AIPromptHistoryCreate fails; do not fail the Update operation
else:
    h.dbUpdate(ctx, ...)  // newInitPrompt is empty; skip pre-fetch and history insert
```

**Temporal constraint:** The pre-fetch `AIGet` call **must** happen before `h.dbUpdate` to read the pre-update value. After `dbUpdate`, the cache holds the new value.

---

## New Handler Package (`bin-ai-manager`)

### `AIPromptHistoryHandler` interface

`pkg/aiprompthistoryhandler/main.go`:

```go
//go:generate mockgen -destination mock_main.go -package aiprompthistoryhandler . AIPromptHistoryHandler

type AIPromptHistoryHandler interface {
    List(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)
    Get(ctx context.Context, aiID uuid.UUID, historyID uuid.UUID) (*aiprompthistory.AIPromptHistory, error)
}
```

**Authorization model:** follows the same pattern as `aihandler.Get` — no explicit `customer_id` parameter. Single-resource gets are scoped by resource ID only; gateway-level isolation assumed. `Get` verifies `history.AIID == aiID`; mismatch returns not-found. This cross-check is enforced in the handler layer — `AIPromptHistoryGet` fetches by `historyID` only.

### Constructor

```go
func New(db dbhandler.DBHandler, util utilhandler.UtilHandler) AIPromptHistoryHandler
```

### Wiring into `listenhandler`

`pkg/listenhandler/main.go` — `listenHandler` struct gains:

```go
aiprompthistoryHandler aiprompthistoryhandler.AIPromptHistoryHandler
```

`NewListenHandler(...)` gains `aiprompthistoryHandler aiprompthistoryhandler.AIPromptHistoryHandler` as a parameter.

`cmd/ai-manager/main.go` — constructs `aiprompthistoryhandler.New(db, util)` and passes it to `NewListenHandler`. No new config flags; `docs/operations.md` and `docs/dependencies.md` do not need updating (no new config, no new external dependencies).

### New route patterns

Following the existing convention (no `^` anchor; list route matches `\?`, single-entry route matches `$`):

```go
regV1AIsIDPromptHistoriesGet   = regexp.MustCompile(`/v1/ais/` + regUUID + `/prompt_histories\?`)
regV1AIsIDPromptHistoriesIDGet = regexp.MustCompile(`/v1/ais/` + regUUID + `/prompt_histories/` + regUUID + `$`)
```

Note: `GET /v1/ais/{id}/prompt_histories` (without `?`) will not match the list pattern and returns 404. This is consistent with the existing behavior of the AI list route — callers must include at least `?size=N`.

### URI parsing

**List route** (uses `url.Parse` — same as `processV1AIsGet`, because URI contains a query string):

```go
u, err := url.Parse(m.URI)
// ai_id: strings.Split(u.Path, "/")[3]
// size, token: u.Query()
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

## DB Layer (`bin-ai-manager`)

### New file: `pkg/dbhandler/aiprompthistory.go`

```go
// AIPromptHistoryCreate inserts a new AIPromptHistory row.
// Sets TMCreate = h.utilHandler.TimeNow() internally (matching all other entity create methods).
func (h *handler) AIPromptHistoryCreate(ctx context.Context, p *aiprompthistory.AIPromptHistory) error

// AIPromptHistoryGet returns a single entry by ID.
// No ai_id parameter — ai_id cross-check is enforced in the handler layer, not the DB layer.
// No cache (infrequent access; out of scope for v1).
func (h *handler) AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error)

// AIPromptHistoryGetsByAIID returns entries for the given AI, newest first.
// Token is a tm_create timestamp string cursor (WHERE tm_create < token ORDER BY tm_create DESC),
// matching the cursor convention used by AIList, MessageList, SummaryList, etc.
// No cache — same rationale as AIPromptHistoryGet.
func (h *handler) AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)
```

---

## API Endpoints

Both endpoints are read-only.

### List prompt history

```
GET /v1/ais/{ai_id}/prompt_histories?size=N&token=T
```

Response: bare JSON array (no envelope wrapper — consistent with all other list endpoints):

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

## Multi-Service Chain

### `bin-common-handler/pkg/requesthandler/ai_ai_prompt_histories.go` (new file)

Following the pattern of `ai_ais.go`:

```go
func (r *requestHandler) AIV1AIPromptHistoryList(ctx context.Context, aiID uuid.UUID, pageToken string, pageSize uint64) ([]*amaiprompthistory.AIPromptHistory, error)
func (r *requestHandler) AIV1AIPromptHistoryGet(ctx context.Context, aiID uuid.UUID, historyID uuid.UUID) (*amaiprompthistory.AIPromptHistory, error)
```

### `bin-openapi-manager/openapi/paths/ais/` (new files)

New YAML spec files for:
- `GET /v1/ais/{ai_id}/prompt_histories` — list operation
- `GET /v1/ais/{ai_id}/prompt_histories/{history_id}` — get operation

After editing the spec, run `go generate ./...` in `bin-openapi-manager`, then rebuild `bin-api-manager` generated types.

### `bin-api-manager` (Two-Level Handler Pattern)

Service layer (`pkg/servicehandler/`):
```go
AIPromptHistoryGetsByAIID(ctx, aiID, pageToken, pageSize) ([]*amaiprompthistory.AIPromptHistory, error)
AIPromptHistoryGet(ctx, aiID, historyID) (*amaiprompthistory.AIPromptHistory, error)
```

HTTP handler (`server/`): calls the service layer, serializes the raw `AIPromptHistory` struct directly. Justified deviation from the `ConvertWebhookMessage()` convention: all `AIPromptHistory` fields are appropriate for external consumers, and no `WebhookMessage` conversion layer exists for this entity. Document this deviation in a comment at the call site.

---

## Error Handling

| Condition | Response |
|-----------|----------|
| No history rows for `ai_id` (list) | empty array `[]` |
| `history_id` not found | `404 Not Found` |
| `history.AIID ≠ ai_id` in URL | `404 Not Found` |
| `init_prompt` empty on create/update | skip history insert silently |
| `init_prompt` unchanged on update | skip history insert silently |
| `AIPromptHistoryCreate` fails | log error, operation succeeds (best-effort side effect) |

---

## Testing

### `pkg/dbhandler/aiprompthistory_test.go`
- Create: inserts row, retrieves by ID, verifies `TMCreate` is set by dbhandler
- Get: returns correct entry; returns error for unknown ID
- GetsByAIID: returns entries in `tm_create DESC` order; pagination (`size`/`token`) works correctly
- GetsByAIID: returns empty slice when no entries exist for given `ai_id`

### `pkg/aiprompthistoryhandler/` (unit tests, gomock, table-driven)
- List: returns entries for valid `ai_id`
- List: returns empty slice when no history rows exist
- Get: returns single entry when `history.AIID == aiID`
- Get: returns not-found error when `history.AIID ≠ aiID`
- Get: returns not-found error when `historyID` does not exist

### `pkg/aihandler/chatbot_test.go` (extend existing)
- Create AI with non-empty `init_prompt` → `AIPromptHistoryCreate` called after `dbCreate`
- Create AI with empty `init_prompt` → `AIPromptHistoryCreate` NOT called
- Update AI with non-empty `init_prompt` differing from current → `AIGet` pre-fetch (before `dbUpdate`), `dbUpdate`, `AIPromptHistoryCreate`
- Update AI with non-empty `init_prompt` same as current → `AIGet` pre-fetch, `dbUpdate`; `AIPromptHistoryCreate` NOT called
- Update AI with empty `init_prompt` → `dbUpdate` only; no `AIGet` pre-fetch, no `AIPromptHistoryCreate`
- Create AI where `AIPromptHistoryCreate` fails → `Create` still returns success; error is logged

### `pkg/listenhandler/` (new `v1_ai_prompt_histories_test.go`)
- List: happy path returns `200` with JSON array
- List: invalid `ai_id` UUID in path returns `400`
- Get: happy path returns `200` with single entry
- Get: invalid `ai_id` or `history_id` UUID returns `400`
- Get: not-found from handler returns appropriate error response

---

## Restore Flow (no new endpoint)

1. `GET /v1/ais/{ai_id}/prompt_histories?size=50` — find desired version
2. Copy the `prompt` field value
3. `PUT /v1/ais/{ai_id}` with the full AI update body including `"init_prompt": "<copied value>"` — triggers a new history entry

Note: the update verb is `PUT` (not `PATCH`) — this service uses `PUT` for AI updates.

---

## Service Docs to Update

| Doc | Update needed |
|-----|---------------|
| `bin-ai-manager/docs/architecture.md` | Add two new route entries |
| `bin-ai-manager/docs/domain.md` | Add `AIPromptHistory` entity description |
| `bin-ai-manager/docs/operations.md` | No change (no new config flags) |
| `bin-ai-manager/docs/dependencies.md` | No change (no new external dependencies) |
| `bin-api-manager/docsdev/source/` | Add RST for new endpoints and `AIPromptHistory` struct; document raw model struct fields directly (no WebhookMessage layer); rebuild HTML; force-add build output |

---

## Out of Scope

- User-level attribution (who changed the prompt) — not needed for v1
- Dedicated restore endpoint — client-side copy is sufficient
- Automatic pruning / retention policy — history is kept indefinitely
- Diff view between versions — presentation concern, handled by clients
- Webhook events for prompt history changes — no operational need
- Cache-aside for history DB reads — infrequent access does not justify extending cachehandler scope
