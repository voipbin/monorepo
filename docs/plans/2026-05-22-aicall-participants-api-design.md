# AIcall Participants API Design

## Goal

Expose the `ai_aicall_participants` table (written by PR #934) via two paginated read endpoints: one scoped to an aicall, one scoped to an AI agent.

## Background

PR #934 added write-side infrastructure that records which AI agents participate in each AIcall. The data is in the DB but has no read path. This feature adds the external API so consumers (e.g., the upcoming AI evaluation feature) can query participation history.

## Endpoints

```
GET /v1/aicalls/{id}/participants?page_size=N&page_token=T
GET /v1/ais/{id}/participants?page_size=N&page_token=T
```

Both return a paginated JSON response to the HTTP caller using the standard `CommonPagination` envelope (assembled at the `bin-api-manager` server layer via `GenerateListResponse`, same shape as `/v1/aicalls` and `/v1/messages`). Note: over the internal RPC hop between `bin-api-manager` and `bin-ai-manager`, the listenhandler returns a **bare JSON array** (no envelope); the `bin-common-handler` reqHandler unmarshals this with `parseResponse(tmp, &res)` into `[]participant.Participant`.

```json
{
  "result": [
    {
      "ai_id":     "22222222-2222-2222-2222-222222222222",
      "aicall_id": "11111111-1111-1111-1111-111111111111",
      "tm_create": "2026-05-22T10:00:00.000000Z"
    }
  ],
  "next_page_token": "2026-05-22T10:00:00.000000Z"
}
```

Pagination follows the existing cursor pattern used by messages, summaries, and aicalls:
- `page_token` is a `tm_create` timestamp (ISO 8601); results before that token are returned. Empty `page_token` defaults to `TimeGetCurTime()` in the dbhandler (matching `MessageList`).
- `page_size` defaults to 100, max 100 (matching `GetAicalls`/`GetAis` in `server/aicalls.go` and `server/ais.go`)
- `next_page_token` is computed in the server handler from the last item's `TMCreate` (same as `server/aicalls.go`). It is absent when the result set is empty.

## Data Model

A new `models/participant/` package in `bin-ai-manager`.

**`models/participant/participant.go`:**

```go
// Participant is a single row from ai_aicall_participants.
// UUID fields use the ,uuid db tag so commondatabasehandler.ScanRow converts BINARY(16) correctly.
type Participant struct {
    AIID     uuid.UUID  `json:"ai_id"     db:"ai_id,uuid"`
    AIcallID uuid.UUID  `json:"aicall_id" db:"aicall_id,uuid"`
    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}

// WebhookMessage is the external-facing representation.
// All fields are public (no internal fields to strip), but ConvertWebhookMessage
// is provided for consistency with the two-level handler pattern.
type WebhookMessage struct {
    AIID     uuid.UUID  `json:"ai_id"`
    AIcallID uuid.UUID  `json:"aicall_id"`
    TMCreate *time.Time `json:"tm_create"`
}

func (p *Participant) ConvertWebhookMessage() *WebhookMessage {
    return &WebhookMessage{
        AIID:     p.AIID,
        AIcallID: p.AIcallID,
        TMCreate: p.TMCreate,
    }
}
```

Note: the existing write path in `pkg/dbhandler/participant.go` calls `.Bytes()` manually for `ai_id`/`aicall_id`. The new read path uses `commondatabasehandler.ScanRow` (reflection-based), which requires the `,uuid` tag.

Note: `Participant` deliberately does NOT embed `identity.Identity`. Monorepo convention requires `identity.Identity` for standard entities, but `ai_aicall_participants` is a composite-key join row (no separate `id`, no `customer_id`) over a table already created by PR #934. This is an intentional exception.

## Architecture

The change spans four services, always modified top-to-bottom:

```
bin-openapi-manager  (OpenAPI spec — schema first)
       ↓
bin-api-manager      (HTTP gateway: server/ + servicehandler/)
       ↓
bin-common-handler   (shared RPC layer: requesthandler/)
       ↓
bin-ai-manager       (listenhandler/ + participanthandler/ + dbhandler/ + models/)
```

### bin-openapi-manager

**New path files:**
- `openapi/paths/aicalls/id_participants.yaml` — `GET /aicalls/{id}/participants` spec: reuse shared `$ref: '#/components/parameters/PageSize'` and `PageToken` query params; response uses `allOf: [CommonPagination, {type: object, properties: {result: {type: array, items: {$ref: '#/components/schemas/AIManagerParticipant'}}}}]` (same shape as `aicalls/main.yaml`).
- `openapi/paths/ais/id_participants.yaml` — `GET /ais/{id}/participants` spec, same shape.

**Modified `openapi/openapi.yaml`:**
- Add the two new path `$ref` entries under `paths:` (matching the pattern at lines 7241/7246 — note: NO `/v1` prefix; the server base URL `https://api.voipbin.net/v1.0` already supplies the prefix):
  ```yaml
  /aicalls/{id}/participants:
    $ref: './paths/aicalls/id_participants.yaml'
  /ais/{id}/participants:
    $ref: './paths/ais/id_participants.yaml'
  ```
- Add `AIManagerParticipant` schema component: `{ ai_id: string (UUID), aicall_id: string (UUID), tm_create: string (datetime) }`.

**Generation:** Run `go generate ./...` in `bin-openapi-manager` first; then re-run `go generate ./...` in `bin-api-manager` to sync `gens/openapi_server/gen.go`.

### bin-ai-manager

**New files:**
- `models/participant/participant.go` — `Participant` struct + `WebhookMessage` + `ConvertWebhookMessage()` (see Data Model section).

**Modified files:**

- `pkg/dbhandler/main.go` — add to `DBHandler` interface:
  ```go
  ParticipantListByAIcallID(ctx context.Context, aicallID uuid.UUID, size uint64, token string) ([]*participant.Participant, error)
  ParticipantListByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*participant.Participant, error)
  ```
  Arg order matches `ParticipantCreate(ctx, aicallID, aiID)` — `aicallID` before `aiID`.

- `pkg/dbhandler/participant.go` — implement both methods. Pattern mirrors `MessageList`:
  ```go
  func (h *handler) ParticipantListByAIcallID(ctx context.Context, aicallID uuid.UUID, size uint64, token string) ([]*participant.Participant, error) {
      if token == "" {
          token = h.utilHandler.TimeGetCurTime()
      }
      cols := commondatabasehandler.GetDBFields(participant.Participant{})
      query, args, err := sq.Select(cols...).
          From(participantTable).
          Where(sq.Eq{"aicall_id": aicallID.Bytes()}).
          Where(sq.Lt{"tm_create": token}).
          OrderBy("tm_create desc").
          Limit(size).
          ToSql()
      // ... scan rows into []*participant.Participant
  }
  ```
  `ParticipantListByAIID` is the symmetric counterpart (`WHERE ai_id = ?`).

- `pkg/participanthandler/main.go` — add to `ParticipantHandler` interface and implementation:
  ```go
  ListByAIcallID(ctx context.Context, aicallID uuid.UUID, size uint64, token string) ([]*participant.Participant, error)
  ListByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*participant.Participant, error)
  ```
  Both delegate directly to dbhandler (no cache — participant rows are immutable once written).

- `pkg/listenhandler/main.go`:
  - Add `participantHandler participanthandler.ParticipantHandler` field to `listenHandler` struct.
  - Extend `NewListenHandler` signature with a `participantHandler participanthandler.ParticipantHandler` arg (positional, after existing args).
  - Add two new regex patterns. Anchor with `(\?|$)` — this matches both the no-query-string form (`/participants` at end of URI) and the form with pagination params (`/participants?page_size=100`), while preventing false matches on future routes like `/participants_xyz`. Do NOT leave the pattern unanchored (prefix-hazard) and do NOT use bare `\?` (breaks calls with no query string). Existing list patterns use `\?` only because they always have query params; sub-resource list patterns should use `(\?|$)`:
    ```go
    regV1AIcallsIDParticipants = regexp.MustCompile("/v1/aicalls/" + regUUID + `/participants(\?|$)`)
    regV1AIsIDParticipants     = regexp.MustCompile("/v1/ais/" + regUUID + `/participants(\?|$)`)
    ```
  - Add two new `case` entries in the dispatch switch.
  - Update `bin-ai-manager/docs/architecture.md` routing table in the same commit.

- `pkg/listenhandler/v1_aicalls.go` — add `processV1AIcallsIDParticipantsGet`: extract aicall UUID from `uriItems[3]`, parse `page_size`/`page_token` from query, call `participantHandler.ListByAIcallID`, JSON-marshal and return.

- `pkg/listenhandler/v1_ais.go` — add `processV1AIsIDParticipantsGet`: same shape; AI UUID from `uriItems[3]`, call `participantHandler.ListByAIID`.

- `cmd/ai-manager/main.go` — two-part wiring change:
  1. Add `participantHandler participanthandler.ParticipantHandler` parameter to `runListen()` function signature (around line 175). `participantHandler` is declared in `run()` scope (around line 120) but `runListen()` is a separate function — it cannot see `run()`'s locals.
  2. Update the `runListen(...)` call site inside `run()` (around line 126) to pass `participantHandler` as the additional argument.
  3. Inside `runListen()`, pass `participantHandler` to the `NewListenHandler(...)` call (around line 186).
  Also update `bin-ai-manager/docs/domain.md` to mention the `Participant` model.

### bin-common-handler

**New file:**
- `pkg/requesthandler/ai_participants.go` — two RPC methods:
  ```go
  func (r *requestHandler) AIV1AIcallParticipantList(ctx context.Context, aicallID uuid.UUID, pageToken string, pageSize uint64) ([]participant.Participant, error) {
      uri := fmt.Sprintf("/v1/aicalls/%s/participants?page_token=%s&page_size=%d", aicallID, url.QueryEscape(pageToken), pageSize)
      // action string follows the uniform <...-id> placeholder convention used across all 30+ AI RPC calls
      tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls/<aicall-id>/participants", requestTimeoutDefault, 0, ContentTypeJSON, nil)
      // parse with shared parseResponse helper: json.Unmarshal(tmp.Data, &res)
      // Note: ai-manager listenhandler returns a BARE JSON array (not CommonPagination envelope);
      // the {result, next_page_token} envelope is assembled only at the bin-api-manager server layer.
      var res []participant.Participant
      if err := parseResponse(tmp, &res); err != nil { return nil, err }
      return res, nil
  }

  func (r *requestHandler) AIV1AIParticipantList(ctx context.Context, aiID uuid.UUID, pageToken string, pageSize uint64) ([]participant.Participant, error) {
      uri := fmt.Sprintf("/v1/ais/%s/participants?page_token=%s&page_size=%d", aiID, url.QueryEscape(pageToken), pageSize)
      tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais/<ai-id>/participants", requestTimeoutDefault, 0, ContentTypeJSON, nil)
      var res []participant.Participant
      if err := parseResponse(tmp, &res); err != nil { return nil, err }
      return res, nil
  }
  ```
  Return type is `[]participant.Participant` (value slice), matching the convention of `AIV1AIcallList` (`ai_aicalls.go:52`).

**Modified file:**
- `pkg/requesthandler/main.go` — add both methods to `RequestHandler` interface.

### bin-api-manager

**New file:**
- `pkg/servicehandler/participant.go` — two service handler methods:
  ```go
  func (h *serviceHandler) AIcallParticipantGets(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID, size uint64, token string) ([]*amparticipant.WebhookMessage, error) {
      // 1. Fetch aicall to get customerID (uses existing private aicallGet helper)
      aicall, err := h.aicallGet(ctx, aicallID)
      // 2. Permission check — matches AIcallGet exactly (pkg/servicehandler/aicall.go:203-209)
      switch {
      case a.IsAgent() || a.IsAccesskey():
          if !h.hasPermission(ctx, a, aicall.CustomerID, PermissionCustomerAdmin|PermissionCustomerManager) { ... }
      case a.IsDirect():
          // HasAllowedResourceType("aicall") — there is no "ai" direct resource type
          if !a.HasAllowedResourceType("aicall") { ... }
          if aicall.CustomerID != a.CustomerID { ... }  // same as AIcallGet: customer-level scope, not resource-ID scope
      }
      // 3. RPC call
      tmps, err := h.reqHandler.AIV1AIcallParticipantList(ctx, aicallID, token, size)
      // 4. Convert (range over value slice; &f auto-takes address of copy for pointer receiver)
      res := make([]*amparticipant.WebhookMessage, 0, len(tmps))
      for _, f := range tmps {
          res = append(res, f.ConvertWebhookMessage())
      }
      return res, nil
  }

  func (h *serviceHandler) AIParticipantGets(...) ([]*amparticipant.WebhookMessage, error) {
      // Direct tokens: return ErrDirectAccessNotSupported — ALL AI resource methods in
      // pkg/servicehandler/ai.go uniformly reject direct tokens (lines 52, 121, 156, 232, 260, 312).
      // AIParticipantGets follows this same pattern for consistency.
      if a.IsDirect() {
          return nil, ErrDirectAccessNotSupported
      }
      // Agent/Accesskey: fetch AI via existing aiGet helper, then check
      // PermissionCustomerAdmin | PermissionCustomerManager on ai.CustomerID.
  }
  ```

**Modified files:**
- `pkg/servicehandler/main.go` — add both methods to `ServiceHandler` interface.
- `server/aicalls.go` — add `GetAicallsIdParticipants(c *gin.Context, id string, params GetAicallsIdParticipantsParams)`: parse UUID, resolve page size (default 100, max 100 — same logic as `GetAicalls` lines 66-72), call servicehandler, use `GenerateListResponse` to wrap with `next_page_token` computed from last item's `TMCreate`.
- `server/ais.go` — add `GetAisIdParticipants`: same shape.

## Permission Model

The two endpoints have different permission models:

**`AIcallParticipantGets` (scoped to aicall):**
1. Fetch aicall without auth check (`aicallGet` private helper).
2. Check caller — mirrors `AIcallGet` exactly (`pkg/servicehandler/aicall.go:203-209`):
   - **Agent/Accesskey**: `PermissionCustomerAdmin | PermissionCustomerManager` on `aicall.CustomerID`.
   - **Direct token**: `a.HasAllowedResourceType("aicall")` then `aicall.CustomerID != a.CustomerID`. Direct tokens for the "aicall" resource type are customer-scoped, not resource-ID-scoped — matches the existing `AIcallGet` check. (No `"ai"` direct resource type exists; `boot.go:20-23` maps `dmdirect.ResourceTypeAI → {"aicall"}`.

**`AIParticipantGets` (scoped to AI agent):**
1. Check caller: **Direct token → return `ErrDirectAccessNotSupported` immediately.** All AI resource methods in `pkg/servicehandler/ai.go` uniformly reject direct tokens — `AIParticipantGets` must match.
2. Fetch AI without auth check (`aiGet` private helper).
3. **Agent/Accesskey**: `PermissionCustomerAdmin | PermissionCustomerManager` on `ai.CustomerID`.

## Error Handling

- Invalid UUID in path → 400
- Parent resource not found → 404 (propagated from `aicallGet`/`aiGet`)
- Permission denied → 403
- DB error → 500 (logged, not exposed to caller)

## Mock Regeneration and Verification

Interface changes in four packages require mock regeneration via `go generate ./...` **before** tests will compile. Run in dependency order:

1. `bin-openapi-manager` — regenerate OpenAPI types (creates new handler signatures in `bin-api-manager/gens/`)
2. `bin-api-manager` — regenerate `gens/openapi_server/gen.go` and `pkg/servicehandler/mock_main.go`
3. `bin-common-handler` — regenerate `pkg/requesthandler/mock_main.go`
4. `bin-ai-manager` — regenerate `pkg/dbhandler/mock_*.go`, `pkg/participanthandler/mock_main.go`, `pkg/listenhandler/mock_main.go`

Then run the full 5-step verification (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`) in each of those four services.

## Testing

Each layer gets table-driven unit tests following the existing gomock pattern:

- `bin-ai-manager/pkg/dbhandler/participant_test.go` — add list tests using the shared in-memory SQLite DB; test empty-token default, pagination, and filter by aicall_id / ai_id
- `bin-ai-manager/pkg/participanthandler/` — mock dbhandler, test list delegation and arg pass-through
- `bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go` / `v1_ais_test.go` — mock participanthandler, test UUID extraction, pagination param parsing, 400 on bad UUID
- `bin-api-manager/pkg/servicehandler/participant_test.go` — mock reqHandler, test permission checks for Agent/Accesskey/Direct caller types; test 403 path
- `bin-api-manager/server/aicalls_test.go` / `server/ais_test.go` — test HTTP handler wiring, `next_page_token` computation, page size clamping

## RST Docs

Per the monorepo CLAUDE.md mandate, add a participant struct entry to the aicall RST docs:
- Add `aicall_struct_participant.rst` (or inline in existing `aicall_struct_*.rst`) documenting the three fields: `ai_id`, `aicall_id`, `tm_create`.
- Update `aicall_overview.rst` to reference the participants sub-resource.
- Clean rebuild: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`
- Force-add: `git add -f bin-api-manager/docsdev/build/`

## Service Docs Sync

Changing `pkg/listenhandler/main.go` (new routes) and `models/participant/participant.go` (new model) in `bin-ai-manager` requires updating:
- `bin-ai-manager/docs/architecture.md` — add two new routes to the routing table
- `bin-ai-manager/docs/domain.md` — add `Participant` to the domain entities section

## Out of Scope

- Write endpoints (participants are written implicitly at call start / team member switch — no external create/delete)
- Returning full AI or AIcall objects (caller fetches details via existing endpoints)
