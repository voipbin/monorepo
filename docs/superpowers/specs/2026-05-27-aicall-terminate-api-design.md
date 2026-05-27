# Design: POST /aicalls/{id}/terminate External API

**Date:** 2026-05-27
**Status:** Draft v3 (post-review)
**Branch:** NOJIRA-Add-aicall-terminate-api

---

## Problem Statement

The AI call terminate action exists internally (RabbitMQ handler `AIV1AIcallTerminate` in
`bin-common-handler`) and in the OpenAPI spec (`/aicalls/{id}/terminate`), but the
`bin-api-manager` service has never been wired up to expose it externally. Customers cannot
terminate an in-progress AI call via the REST API.

---

## Current State

| Layer | Status | Location |
|---|---|---|
| OpenAPI spec (`id_terminate.yaml`) | ✅ Exists on `main` | `bin-openapi-manager/openapi/paths/aicalls/` |
| Route in `openapi.yaml` | ✅ Registered on `main` | `bin-openapi-manager/openapi/openapi.yaml` |
| RabbitMQ request handler (`AIV1AIcallTerminate`) | ✅ Implemented | `bin-common-handler/pkg/requesthandler/ai_aicalls.go` |
| Requesthandler mock (`AIV1AIcallTerminate`) | ✅ Already present | `bin-common-handler/pkg/requesthandler/mock_main.go` |
| Generated server interface (`gen.go`) | ❌ Not regenerated — `PostAicallsIdTerminate` absent | `bin-api-manager/gens/openapi_server/gen.go` |
| Service handler interface + impl | ❌ Missing | `bin-api-manager/pkg/servicehandler/` |
| Service handler mock | ❌ Not regenerated | `bin-api-manager/pkg/servicehandler/mock_main.go` |
| HTTP server handler | ❌ Missing | `bin-api-manager/server/aicalls.go` |
| Tests | ❌ Missing | — |
| RST docs + routing.md | ❌ Missing | `bin-api-manager/docs/`, `docsdev/source/` |

**Note:** `bin-common-handler` needs **no changes** — the RabbitMQ request handler and its
mock are fully implemented. All work is isolated to `bin-api-manager`.

---

## Design

### Endpoint

```
POST /aicalls/{id}/terminate
```

- **Authentication:** JWT (agent/accesskey) or direct token
- **Authorization:** Same as `DELETE /aicalls/{id}` (see Permission Model below)
- **Request body:** None
- **Success response:** `200 OK` with `AIManagerAIcall` JSON body (the terminated aicall)
- **Error responses:** `400`, `401`, `403`, `404`, `500` (already defined in the OpenAPI spec)

### Permission Model

Mirrors `AIcallDelete` exactly:

| Auth type | Required permission |
|---|---|
| Agent or Accesskey | `PermissionCustomerAdmin` or `PermissionCustomerManager` |
| Direct token | Resource type `"aicall"` must be in allowed list AND `aicall.CustomerID == token.CustomerID` |

### Data Flow

```
Client
  → POST /aicalls/{id}/terminate
  → bin-api-manager: PostAicallsIdTerminate (server/aicalls.go)
    → serviceHandler.AIcallTerminate (pkg/servicehandler/aicall.go)
      → aicallGet (ownership / existence verification)
      → hasPermission / direct-token check (auth check)
      → reqHandler.AIV1AIcallTerminate (bin-common-handler)
        → RabbitMQ RPC → bin-ai-manager: POST /v1/aicalls/{id}/terminate
  ← 200 OK: aicall.ConvertWebhookMessage()
```

---

## Implementation Steps

### Step 1: Regenerate `bin-api-manager` server code [REQUIRED]

**This step is required before writing any code in Steps 4–5.** `PostAicallsIdTerminate` does
not yet appear in `gen.go` — the method is absent from the `ServerInterface` and the Gin
router. Without regeneration the package will not compile.

```bash
# Regenerate the types bundle in bin-openapi-manager (produces gens/models/gen.go)
cd bin-openapi-manager && go generate ./... && go build ./...

# Regenerate the server interface in bin-api-manager (produces gens/openapi_server/gen.go)
# This adds PostAicallsIdTerminate to ServerInterface and registers the Gin route.
cd bin-api-manager && go generate ./...
```

After Step 1, `gens/openapi_server/gen.go` will contain:
```go
PostAicallsIdTerminate(c *gin.Context, id openapi_types.UUID)
```
and the route `POST /aicalls/:id/terminate` registered in the router.

**Verify the actual generated signature before coding Step 5.** The `format: uuid` annotation
in `id_terminate.yaml` causes the generator to emit `openapi_types.UUID` (same as
`GetAicallsIdParticipants`), but confirm this after generation.

### Step 2: Service handler interface (`pkg/servicehandler/main.go`)

Add to the `ServiceHandler` interface (near the other `AIcall*` methods, around line 318):

```go
AIcallTerminate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)
```

### Step 3: Regenerate service handler mock [REQUIRED]

After adding `AIcallTerminate` to the interface, regenerate the mock so tests compile:

```bash
cd bin-api-manager && go generate ./...
```

This updates `pkg/servicehandler/mock_main.go` with `EXPECT().AIcallTerminate(...)`.

### Step 4: Service handler implementation (`pkg/servicehandler/aicall.go`)

Append after `AIcallDelete`, mirroring it exactly:

```go
// AIcallTerminate terminates the aicall.
func (h *serviceHandler) AIcallTerminate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error) {
    c, err := h.aicallGet(ctx, id)
    if err != nil {
        return nil, errors.Wrapf(err, "could not get aicall info")
    }

    switch {
    case a.IsAgent() || a.IsAccesskey():
        if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
            return nil, serviceerrors.ErrPermissionDenied
        }
    case a.IsDirect():
        if !a.HasAllowedResourceType("aicall") {
            return nil, fmt.Errorf("%w: direct token does not allow this resource type", serviceerrors.ErrPermissionDenied)
        }
        if c.CustomerID != a.CustomerID {
            return nil, fmt.Errorf("%w: resource not in token scope", serviceerrors.ErrPermissionDenied)
        }
    }

    tmp, err := h.reqHandler.AIV1AIcallTerminate(ctx, id)
    if err != nil {
        return nil, errors.Wrapf(err, "could not terminate the aicall")
    }

    res := tmp.ConvertWebhookMessage()
    return res, nil
}
```

### Step 5: HTTP server handler (`server/aicalls.go`)

Append after `DeleteAicallsId`.

**Signature note:** The generated interface uses `openapi_types.UUID` (not `string`) because
`id_terminate.yaml` specifies `format: uuid`. This differs from `DeleteAicallsId` and
`GetAicallsId` (whose `id.yaml` spec lacks `format: uuid` and therefore generates `string`).
Because `openapi_types.UUID` is a `[16]byte` value type (never nil), the nil-UUID guard
(`if target == uuid.Nil`) present in the `string`-based handlers is not needed here.

```go
func (h *server) PostAicallsIdTerminate(c *gin.Context, id openapi_types.UUID) {
    log := logrus.WithFields(logrus.Fields{
        "func":            "PostAicallsIdTerminate",
        "request_address": c.ClientIP(),
    })

    a, ok := getAuthIdentity(c)
    if !ok {
        log.Errorf("Could not find auth identity.")
        abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
        return
    }
    log = log.WithFields(logrus.Fields{"auth": a})

    target := uuid.UUID(id)
    res, err := h.serviceHandler.AIcallTerminate(c.Request.Context(), a, target)
    if err != nil {
        log.Errorf("Could not terminate the aicall. err: %v", err)
        abortWithServiceError(c, err)
        return
    }

    c.JSON(200, res)
}
```

Note: use `c.ClientIP()` (with parentheses) — the method call, not the method value. Older
handlers in `aicalls.go` omit the `()` (a latent bug that logs a hex function pointer
instead of the IP string); do not copy that pattern. File a separate follow-up to fix all
affected handlers in `aicalls.go`.

### Step 6: Tests

#### `server/aicalls_test.go` — `TestPostAicallsIdTerminate`

| Case | Mock setup | Expected HTTP |
|---|---|---|
| Happy path | `AIcallTerminate` returns valid `WebhookMessage` | `200 OK` |
| Service error | `AIcallTerminate` returns permission error | `403 Forbidden` |

#### `pkg/servicehandler/aicall_test.go` — `Test_AIcallTerminate`

Mock the requesthandler interface (`mockReq`), not the unexported `aicallGet` helper.
`aicallGet` calls `AIV1AIcallGet` internally — mock that.

| Case | Mock setup | Expected |
|---|---|---|
| Normal (agent, CustomerAdmin) | `mockReq.AIV1AIcallGet` succeeds, `hasPermission` grants, `mockReq.AIV1AIcallTerminate` succeeds | `WebhookMessage` returned |
| `aicallGet` failure | `mockReq.AIV1AIcallGet` returns error | error propagated |
| Agent permission denied | `hasPermission` returns false | `ErrPermissionDenied` |
| Direct token — resource type not allowed | `HasAllowedResourceType("aicall")` returns false | `ErrPermissionDenied` |
| Direct token — customer ID mismatch | `aicall.CustomerID != token.CustomerID` | `ErrPermissionDenied` |
| RPC failure | `mockReq.AIV1AIcallGet` succeeds, `mockReq.AIV1AIcallTerminate` returns error | error propagated |

### Step 7: Docs

#### `bin-api-manager/docs/routing.md`

First check whether `GET /aicalls/:id/participants` is already present in the file — it
may have been added in a subsequent commit after `NOJIRA-Add-aicall-participants-api`. Add
only the rows that are missing; do not duplicate. After `DELETE /aicalls/:id`, the two
potentially missing rows are:

```
| GET    | `/aicalls/:id/participants` | bin-ai-manager | List AI call participants |
| POST   | `/aicalls/:id/terminate`   | bin-ai-manager | Terminate AI call session |
```

#### `bin-api-manager/docsdev/source/ai_overview.rst`

Add a `POST /aicalls/{id}/terminate` subsection. The existing AI Participants subsection
(around line 801 in `ai_overview.rst`) is the structural reference — follow the heading,
URL line, and path parameter description. Omit the query parameters table (terminate has
none); focus on the action semantics (terminates an in-progress AI call) and the response
schema.

Rebuild HTML after editing:

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```

### Step 8: api-validator test

Per project convention, add a test to `~/gitvoipbin/monorepo-monitoring/api-validator/`
that calls `POST /aicalls/{non-existent-id}/terminate`. This confirms the endpoint is wired
without triggering live state change. Assert on the error code (`RESOURCE_NOT_FOUND`) in the
response body rather than HTTP status alone, since the error translation chain produces the
status via the typed `VoipbinError` path.

---

## Commit Strategy

All changes go in a single commit on the feature branch (squash-merged to `main`):
- generated files (`gen.go`, `mock_main.go`)
- service handler interface + implementation
- server handler
- tests
- `routing.md` update
- RST source + rebuilt HTML (`docsdev/build/`)

Do not commit the implementation before running `go generate ./...` — the package will not
compile without the generated interface.

---

## Verification

Run the full 5-step workflow in `bin-api-manager` before committing:

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

`bin-common-handler` requires no changes and no re-verification.
`bin-openapi-manager`: run `go build ./...` after generating. If `go generate ./...`
produces no diff in `gens/models/gen.go` (expected — the spec already existed on `main`),
no commit and no full 5-step workflow is needed there.

---

## Out of Scope

- Changes to `bin-ai-manager` (terminate logic already exists there)
- Changes to `bin-common-handler` (request handler and mock already exist — no action needed)
- New billing events or webhook message fields
- Changes to any service other than `bin-api-manager`
