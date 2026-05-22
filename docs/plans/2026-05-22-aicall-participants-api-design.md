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

Both return a paginated JSON array of participant rows:

```json
[
  {
    "ai_id":     "22222222-2222-2222-2222-222222222222",
    "aicall_id": "11111111-1111-1111-1111-111111111111",
    "tm_create": "2026-05-22T10:00:00.000000Z"
  }
]
```

Pagination follows the existing cursor pattern used by messages, summaries, and aicalls:
- `page_token` is a `tm_create` timestamp (ISO 8601); results before that token are returned
- `page_size` defaults to 20, max 100
- Response includes a `next_page_token` field (last item's `tm_create`) when more results exist

## Data Model

A new `models/participant/` package in `bin-ai-manager`:

```go
// Participant is a single row from ai_aicall_participants.
type Participant struct {
    AIID     uuid.UUID  `json:"ai_id"     db:"ai_id"`
    AIcallID uuid.UUID  `json:"aicall_id" db:"aicall_id"`
    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}

// WebhookMessage is the external-facing representation (all fields are public,
// identical to Participant — no internal fields to strip).
type WebhookMessage struct {
    AIID     uuid.UUID  `json:"ai_id"`
    AIcallID uuid.UUID  `json:"aicall_id"`
    TMCreate *time.Time `json:"tm_create"`
}

func (p *Participant) ConvertWebhookMessage() *WebhookMessage { ... }
```

## Architecture

The change spans five locations, always modified top-to-bottom:

```
bin-openapi-manager  (OpenAPI spec — schema first)
       ↓
bin-api-manager      (HTTP gateway: server/ + servicehandler/)
       ↓
bin-common-handler   (shared RPC layer: requesthandler/)
       ↓
bin-ai-manager       (listenhandler/ + participanthandler/ + dbhandler/)
```

### bin-openapi-manager

New path files:
- `openapi/paths/aicalls/id_participants.yaml` — `GET /aicalls/{id}/participants`
- `openapi/paths/ais/id_participants.yaml` — `GET /ais/{id}/participants`

New schema component in `openapi/openapi.yaml`:
- `AIManagerParticipant` — `{ ai_id: string, aicall_id: string, tm_create: string }`

Run `go generate ./...` here first; then re-run in `bin-api-manager` to sync `gens/openapi_server/gen.go`.

### bin-ai-manager

**New files:**
- `models/participant/participant.go` — `Participant` struct + `WebhookMessage` + `ConvertWebhookMessage()`

**Modified files:**
- `pkg/dbhandler/main.go` — add `ParticipantListByAIcallID` and `ParticipantListByAIID` to `DBHandler` interface
- `pkg/dbhandler/participant.go` — implement both: cursor-paginated `SELECT` with `WHERE aicall_id=?` / `WHERE ai_id=?`, `tm_create < token`, `ORDER BY tm_create DESC LIMIT size`
- `pkg/participanthandler/main.go` — add `ListByAIcallID` and `ListByAIID` to interface and implementation (delegates to dbhandler)
- `pkg/listenhandler/main.go` — add `participantHandler` field to `listenHandler` struct; add two new regex patterns and route cases
- `pkg/listenhandler/v1_aicalls.go` — add `processV1AIcallsIDParticipantsGet`: extract UUID from URL, parse pagination params, call `participantHandler.ListByAIcallID`, return JSON
- `pkg/listenhandler/v1_ais.go` — add `processV1AIsIDParticipantsGet`: same shape for AI variant
- `cmd/ai-manager/main.go` — pass `participantHandler` into `listenHandler` constructor

### bin-common-handler

**New file:**
- `pkg/requesthandler/ai_participants.go` — `AIV1AIcallParticipantList` and `AIV1AIParticipantList`: build URI with pagination params, call `sendRequestAI`, parse response

**Modified file:**
- `pkg/requesthandler/main.go` — add both methods to `RequestHandler` interface

### bin-api-manager

**New file:**
- `pkg/servicehandler/participant.go` — `AIcallParticipantGets` and `AIParticipantGets`:
  - Fetch parent resource (aicall or AI) to resolve `customerID`
  - Check `PermissionCustomerAdmin|PermissionCustomerManager` (or direct token scope)
  - Call `reqHandler.AIV1AIcallParticipantList` / `AIV1AIParticipantList`
  - Convert each result via `ConvertWebhookMessage()`

**Modified files:**
- `pkg/servicehandler/main.go` — add both methods to `ServiceHandler` interface
- `server/aicalls.go` — add `GetAicallsIdParticipants`: parse id + pagination, call servicehandler, return list response
- `server/ais.go` — add `GetAisIdParticipants`: same shape

## Permission Model

Both endpoints follow the existing two-level pattern:

1. Fetch parent resource without auth check (private helper: `aicallGet` / `aiGet`)
2. Check caller owns that resource:
   - Agent/Accesskey: `PermissionCustomerAdmin | PermissionCustomerManager`
   - Direct token: resource type must include `"aicall"` / `"ai"` respectively and scope must match

## Error Handling

- Invalid UUID in path → 400
- Parent resource not found → 404 (propagated from aicallGet/aiGet)
- Permission denied → 403
- DB error → 500 (logged, not exposed to caller)
- Duplicate inserts are still silently ignored (write side, unchanged)

## Testing

Each layer gets table-driven unit tests following the existing gomock pattern:

- `pkg/dbhandler/participant_test.go` — add list tests using the shared in-memory SQLite DB
- `pkg/participanthandler/` — mock dbhandler, test list delegation
- `pkg/listenhandler/v1_aicalls_test.go` / `v1_ais_test.go` — mock participanthandler, test HTTP routing and pagination parsing
- `pkg/servicehandler/participant_test.go` (bin-api-manager) — mock reqHandler, test permission checks
- `server/aicalls_test.go` / `server/ais_test.go` (bin-api-manager) — test HTTP handler wiring

## Out of Scope

- Write endpoints (participants are written implicitly at call start / team member switch — no external create/delete)
- Returning full AI or AIcall objects (caller fetches details via existing endpoints)
- RST docs update — implementer must check whether existing aicall or AI RST pages (in `bin-api-manager/docsdev/source/`) document sub-resources; if a `participants` section is warranted, add it. The Swagger/ReDoc API reference is auto-generated from the OpenAPI spec and requires no manual update.
