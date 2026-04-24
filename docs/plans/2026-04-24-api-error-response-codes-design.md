# VoIPbin API Error Response Codes — Design

- **Date:** 2026-04-24
- **Author:** pchero
- **Status:** Design approved (self-review: 7 rounds, 35 fixes folded in)
- **Target service:** `bin-api-manager` (with shared types in `bin-common-handler`)

---

## 1. Problem statement

Today `bin-api-manager` returns `HTTP 400` with an empty body for ~1047 error sites across 66 handler files. Clients cannot distinguish permission failures, not-found, validation errors, or server failures from each other. Server-side log correlation with client-reported errors is ad-hoc.

## 2. Goal & scope

Replace the current pattern with:
- A proper HTTP status code per error class.
- A JSON response body carrying a VoIPbin reason code, originating domain, human-readable message, and a request correlation ID.

**In scope:** every external HTTP endpoint served by `bin-api-manager`. WebSocket pre-upgrade errors (before the socket handshake completes).

**Out of scope:** WebSocket post-upgrade frame errors (use WebSocket close codes). Internal-manager RPC behavior (hybrid plan allows indefinite coexistence).

## 3. Design decisions (user-validated)

| # | Decision | Choice |
|---|---|---|
| Q1 | HTTP status codes | Fix to proper codes (401/403/404/409/503 + 402/429) **and** add error body |
| Q2 | Error code naming scheme | Google-style flat `UPPER_SNAKE` `reason` with `domain` metadata |
| Q3 | Error origination | Hybrid — shared `VoipbinError` type, api-manager adopts immediately, internal managers migrate gradually |
| Q4 | Response body shape | Minimal envelope: `{error: {status, reason, domain, message, request_id}}` + reserved `details` |
| Q5 | Rollout | Incremental by resource group (~9 migration PRs after foundation) |

## 4. Architecture overview

A shared `VoipbinError` type lives in `bin-common-handler`. `bin-api-manager` uses it for all local errors (permission, validation, auth) starting day one. For RPC errors from internal managers, `bin-api-manager` runs a translator: if the incoming error is already a `VoipbinError` (manager has migrated), pass it through; otherwise fall back through sentinel matching → transport-failure detection → substring matching → `INTERNAL`. A Gin middleware generates a `request_id`, attaches it to the logger and response, and propagates it to downstream RPC. A single `abortWithServiceError` helper replaces all 1047 `c.AbortWithStatus(400)` sites incrementally across 9 resource-grouped PRs plus a foundation split into 0a/0b.

## 5. Shared error type (`bin-common-handler/models/errors`)

```go
type Status string

const (
    StatusInvalidArgument     Status = "INVALID_ARGUMENT"
    StatusUnauthenticated     Status = "UNAUTHENTICATED"
    StatusPaymentRequired     Status = "PAYMENT_REQUIRED"
    StatusPermissionDenied    Status = "PERMISSION_DENIED"
    StatusNotFound            Status = "NOT_FOUND"
    StatusAlreadyExists       Status = "ALREADY_EXISTS"
    StatusFailedPrecondition  Status = "FAILED_PRECONDITION"
    StatusResourceExhausted   Status = "RESOURCE_EXHAUSTED"
    StatusUnavailable         Status = "UNAVAILABLE"
    StatusInternal            Status = "INTERNAL"
)

type VoipbinError struct {
    Status  Status `json:"status"`
    Reason  string `json:"reason"`   // UPPER_SNAKE, domain-scoped
    Domain  string `json:"domain"`   // e.g. "call-manager"
    Message string `json:"message"`
    Cause   error  `json:"-"`        // server-side only, never serialized
}

func (e *VoipbinError) Error() string
func (e *VoipbinError) Unwrap() error           // supports errors.Is / errors.As
func (e *VoipbinError) Wrap(cause error) *VoipbinError

// Constructors — one per canonical status
func InvalidArgument(domain, reason, message string) *VoipbinError
func Unauthenticated(domain, reason, message string) *VoipbinError
func PaymentRequired(domain, reason, message string) *VoipbinError
func PermissionDenied(domain, reason, message string) *VoipbinError
func NotFound(domain, reason, message string) *VoipbinError
func AlreadyExists(domain, reason, message string) *VoipbinError
func FailedPrecondition(domain, reason, message string) *VoipbinError
func ResourceExhausted(domain, reason, message string) *VoipbinError
func Unavailable(domain, reason, message string) *VoipbinError
func Internal(domain, reason, message string) *VoipbinError
```

**Domain naming:** uses `commonoutline.ServiceName*` constants (`"call-manager"`, `"billing-manager"`, `"api-manager"`, etc.). Add `ServiceNameApiManager = "api-manager"` if not already present.

### 5.1 RPC carrier contract (additive, non-breaking)

`sock.Response` already has `{StatusCode, DataType, Data}`. When an internal manager returns an error:

- Set `StatusCode >= 400`.
- Set `DataType = "voipbin_error"`.
- Set `Data = json.Marshal(VoipbinError)`.

`api-manager`'s RPC wrapper detects `DataType == "voipbin_error"` and unmarshals into a typed `*VoipbinError`. Managers that haven't migrated simply don't set this DataType; the fallback path handles them.

### 5.2 Request correlation via `sock.Request.RequestID`

Add optional `RequestID string \`json:"request_id,omitempty"\`` to `sock.Request`. The api-manager middleware stores the ID in `context.WithValue`, the RPC wrapper reads it, populates `sock.Request.RequestID`, and downstream services add it to their logrus fields so inbound/outbound log lines correlate by the same ID.

## 6. Canonical status → HTTP mapping

| Status | HTTP | When |
|---|---|---|
| `INVALID_ARGUMENT` | 400 | malformed request, invalid field, unparseable id |
| `UNAUTHENTICATED` | 401 | missing/invalid JWT or access key |
| `PAYMENT_REQUIRED` | 402 | insufficient balance, suspended billing account |
| `PERMISSION_DENIED` | 403 | authenticated but not authorized |
| `NOT_FOUND` | 404 | resource doesn't exist or belongs to another customer |
| `ALREADY_EXISTS` | 409 | duplicate create (explicit construction only) |
| `FAILED_PRECONDITION` | 409 | resource in wrong state (409 tie-breaker default) |
| `RESOURCE_EXHAUSTED` | 429 | rate limit, quota |
| `UNAVAILABLE` | 503 | upstream manager RPC timeout, RabbitMQ unreachable |
| `INTERNAL` | 500 | unclassified failures, panic recovery |

**409 tie-breaker:** when the translator receives a 409-equivalent failure with no typed error, default to `FAILED_PRECONDITION`. `ALREADY_EXISTS` is only emitted on explicit construction (e.g., duplicate-create handlers).

**Enum is intentionally closed.** New statuses require a coordinated schema bump across consumers. `reason` is open-ended for per-domain extensibility.

**Reason code catalog:** maintained at `bin-api-manager/docsdev/source/restful_api_errors.rst`. Each migration PR appends newly emitted reasons to the table. Reviewers reject PRs that introduce new reasons without cataloguing.

## 7. `bin-api-manager` server-layer changes

### 7.1 Helpers (`server/error.go`)

```go
func abortWithError(c *gin.Context, err *cerrors.VoipbinError)
func abortWithServiceError(c *gin.Context, err error)   // runs translator
func httpStatusFor(s cerrors.Status) int                 // 10-entry switch
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder,
                          status cerrors.Status, reason string)  // test helper
```

### 7.2 Request-ID middleware (`lib/middleware/request_id.go`)

- Generates `req_` + 26-char Crockford-ULID = 30 chars total.
- Echoes inbound `X-Request-Id` if the client sends one.
- Stored in `gin.Context`, `c.Request.Context()`, and logrus fields.
- Always emitted as response header `X-Request-Id`.

### 7.3 Existing middleware migrated in foundation

- `lib/middleware/ratelimit.go` → `ResourceExhausted("api-manager", "RATE_LIMIT_EXCEEDED", ...)`.
- `lib/middleware/authenticate.go` → `Unauthenticated(...)` and `PermissionDenied(...)` paths — replaces bare `c.AbortWithStatus(401)` and the existing `c.AbortWithStatusJSON(403, gin.H{...})`.

### 7.4 Handler pattern (before → after)

```go
// Before (1047 sites)
if err != nil {
    log.Errorf("Could not ...: %v", err)
    c.AbortWithStatus(400)
    return
}

// After
if err != nil {
    abortWithServiceError(c, err)
    return
}
```

Per-site `log.Errorf` calls are removed — the translator emits a single structured log line per error.

### 7.5 Response envelope

```json
{
  "error": {
    "status": "PERMISSION_DENIED",
    "reason": "BILLING_ACCESS_DENIED",
    "domain": "billing-manager",
    "message": "You do not have permission to access this billing account.",
    "request_id": "req_01hxyz..."
  }
}
```

## 8. Translator (`server/error_translate.go`)

Priority chain — first match wins:

1. **Typed passthrough:** `errors.As(err, &*VoipbinError)` → use directly.
2. **Sentinel match:** `errors.Is(err, shandlererrors.ErrX)` → convert with a canned generic message. Sentinels live at `bin-api-manager/pkg/serviceerrors/sentinels.go` (parallel to `servicehandler/`, not nested).
3. **Transport failure detection:** `errors.Is(err, context.Canceled)` → log-only, no body (client is gone). `errors.Is(err, context.DeadlineExceeded)` or RabbitMQ transport errors → `Unavailable("api-manager", "UPSTREAM_UNAVAILABLE", ...)`.
4. **Substring fallback (shrinks over time):** small set of stable legacy patterns — `"no permission"` → `PermissionDenied`, `"not found"` → `NotFound`, `"authentication required"` → `Unauthenticated`, `"unavailable"` → `Unavailable`.
5. **Default:** `Internal("api-manager", "INTERNAL", "An internal error occurred.").Wrap(err)`. Original error wrapped for server-side logs only, never serialized to client.

**Defense:** the entire translator runs inside `defer recover()` — a panic produces a safe `INTERNAL` fallback rather than dropping the response.

**Logging:** one structured line per error with `request_id`, `status`, `reason`, `domain`, `route`, and `cause` (server-side only).

### 8.1 Construction guidance (preference order)

1. **Preferred** — servicehandler constructs typed errors with rich context:
   `cerrors.PermissionDenied("ai-manager", "KNOWLEDGE_BASE_FOREIGN", "Knowledge base does not belong to this customer.")`
2. **Acceptable** — return a sentinel when there's no context to add.
3. **Deprecated** — `fmt.Errorf("...")` legacy; caught by substring fallback. Migration target.

### 8.2 Message-quality rule

Messages may echo the client's own resource IDs and parameter names. Messages must NOT include internal DB IDs, tokens, other customers' IDs, stack traces, internal hostnames, or PII from other users. Code-review checklist item.

## 9. OpenAPI schema (`bin-openapi-manager/openapi/openapi.yaml`)

```yaml
components:
  schemas:
    ErrorBody:
      type: object
      required: [status, reason, domain, message, request_id]
      properties:
        status:
          type: string
          enum: [INVALID_ARGUMENT, UNAUTHENTICATED, PAYMENT_REQUIRED,
                 PERMISSION_DENIED, NOT_FOUND, ALREADY_EXISTS,
                 FAILED_PRECONDITION, RESOURCE_EXHAUSTED, UNAVAILABLE, INTERNAL]
          description: Canonical error status. Maps 1:1 to HTTP status code.
        reason:
          type: string
          description: Specific VoIPbin reason code (UPPER_SNAKE). Open-ended.
          example: CALL_NOT_FOUND
        domain:
          type: string
          description: Originating manager service.
          example: call-manager
        message:
          type: string
          description: Human-readable message for debugging.
        request_id:
          type: string
          description: Request correlation ID. Include in support tickets.
          example: req_01hxyz...
        details:
          type: array
          items:
            type: object
            additionalProperties: true
          description: Reserved for future per-field or structured error detail. May be omitted.
    ErrorResponse:
      type: object
      required: [error]
      properties:
        error: { $ref: '#/components/schemas/ErrorBody' }

  responses:
    BadRequest:       { description: "Invalid request (INVALID_ARGUMENT).",      content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
    Unauthenticated:  { description: "Authentication required (UNAUTHENTICATED).", content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
    PaymentRequired:  { description: "Payment required (PAYMENT_REQUIRED).",     content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
    PermissionDenied: { description: "Insufficient permission (PERMISSION_DENIED).", content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
    NotFound:         { description: "Resource not found (NOT_FOUND).",          content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
    Conflict:         { description: "State conflict (ALREADY_EXISTS or FAILED_PRECONDITION).", content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
    TooManyRequests:  { description: "Rate or quota exceeded (RESOURCE_EXHAUSTED).", content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
    Unavailable:      { description: "Upstream unavailable (UNAVAILABLE).",      content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
    InternalError:    { description: "Internal error (INTERNAL).",               content: { application/json: { schema: { $ref: '#/components/schemas/ErrorResponse' } } } }
```

**Per-endpoint:** each path references only the error responses it actually emits. Default set: 400/401/403/404/500. Billing endpoints add 402. Write endpoints add 409. Rate-limited paths add 429. RPC-heavy paths add 503.

**`details` is reserved (optional) from day one** so field-level validation can be added later without a breaking schema change.

## 10. Rollout

### 10.1 PR 0a — `bin-common-handler` foundation

- `VoipbinError` type + constructors + `Error()`/`Unwrap()`/`Wrap()` + tests.
- `sock.Request.RequestID` optional field.
- RPC-response unmarshal helpers for `DataType = "voipbin_error"`.
- Must land before any api-manager work depends on it.

### 10.2 PR 0b — `bin-api-manager` infrastructure

- `abortWithError` / `abortWithServiceError` / `httpStatusFor` / `assertErrorResponse` helpers + tests.
- Request-ID middleware registered in `cmd/api-manager/main.go`.
- Migrate `ratelimit.go` and `authenticate.go` middleware.
- OpenAPI `ErrorBody` / `ErrorResponse` / named responses components added.
- Update `restful_api.rst` with the new envelope + HTTP-status table. Create `restful_api_errors.rst` catalog page. Rebuild + commit Sphinx HTML.
- **Spike:** wire error responses into ONE trivial endpoint (e.g., `GET /v1.0/ping`), regenerate `gens/openapi_server/gen.go`, confirm `oapi-codegen` output is compatible with the `abortWithError` pattern. If incompatible, pivot to keeping components available for docs-only (no per-path wiring) and document the pivot.
- **Audit:** verify current managers' `Response.StatusCode` usage. Document invariant that `StatusCode >= 400` implies error semantics. Fix violators in cleanup PRs before their migration.

### 10.3 Pre-flight before PR 1 (blocking)

Audit `admin.voipbin.net` and `talk.voipbin.net` frontend repos for hardcoded `status === 400` branches. Coordinate deployment so clients tolerate 401/403/404/409/429/503 before api-manager PR 1 rolls out.

### 10.4 PRs 1–9 — handler migration, grouped by resource

| # | Group | Files (approx) |
|---|---|---|
| 1 | Auth & identity | `auth_*`, `me`, `customer`, `service_agents_*` auth if separate codepath |
| 2 | Calls | `calls`, `groupcalls`, `recordings`, `recordingfiles`, `transfers` |
| 3 | Flows & activeflows | `flows`, `activeflows` |
| 4 | Numbers & providers | `numbers`, `available_numbers`, `providers`, `providercalls`, `trunks`, `routes` |
| 5 | Billing | `billings`, `billing_account`, `billing_accounts` (exercises 402) |
| 6 | Messaging & conversations | `messages`, `emails`, `conversations`, `conversation_accounts` |
| 7 | AI, transcription, speaking | `ais`, `aicalls`, `aimessages`, `aisummaries`, `transcribes`, `transcripts`, `speakings` |
| 8 | Agents, queues, conferences, campaigns | `agents`, `queues`, `queuecalls`, `conferences`, `conferencecalls`, `campaigns`, `campaigncalls`, `outplans`, `outdials` |
| 9 | Storage, extensions, misc | `storage_*`, `extensions`, `tags`, `teams`, `timelines*`, `aggregated_events`, `contacts`, `service_agents_*` (remaining), `ws`, `accesskeys`, `rags` |

Each migration PR:

- Migrates server handler files to `abortWithServiceError`.
- Adds/converts sentinels or constructs typed errors in servicehandler for that resource group.
- Adds OpenAPI `responses` refs to the group's paths, appends new reason codes to the catalog RST.
- Updates handler tests using `assertErrorResponse`.
- Updates `*_overview.rst`, `*_tutorial.rst`, `*_troubleshooting.rst` per AI-native RST rules (cause+fix pairs).
- Companion PR in `monorepo-monitoring` updating api-validator tests for the migrated endpoints.

### 10.5 PR N+1 — shrink fallback

Remove the substring fallback from the translator. Any unmatched error produces a clean `INTERNAL`. Unmatched = bug.

### 10.6 Per-PR verification (mandatory)

```bash
cd bin-openapi-manager && go generate ./... && go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager    && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build && git add -f build
```

## 11. Out of scope

1. **Field-level validation details** — envelope is forward-compatible via the reserved `details` array.
2. **Internal-manager adoption of `VoipbinError`** — hybrid plan allows indefinite coexistence.
3. **Rate-limit infrastructure** — vocabulary exists (`RESOURCE_EXHAUSTED`/429); no new middleware in this effort.
4. **Reason-code versioning** — reasons are append-only; rename/remove requires a future deprecation process.
5. **`doc_url` field** — RST anchor stability not guaranteed.
6. **WebSocket post-upgrade errors** — use WebSocket close codes, not this envelope.
7. **Expanding the canonical enum beyond 10** — closed set; future expansion requires coordinated schema bump.

## 12. Risks & mitigations

| Risk | Mitigation |
|---|---|
| `oapi-codegen` produces incompatible typed response variants | PR 0b spike with one endpoint; pivot to docs-only components if needed |
| Frontend clients break on HTTP status change | Pre-flight audit of admin & agent repos blocks PR 1 |
| Manager `Response.StatusCode` inconsistencies | Audit + cleanup during PR 0a |
| Translator panic | `defer recover()` fallback + unit tests |
| Test churn across 66 handler files | `assertErrorResponse` helper reduces boilerplate; per-PR scope keeps it manageable |
| Reason-code drift | Central RST catalog + code-review checklist |
| PR 0a ↔ 0b coupling forces joint iteration | Expect some rework of 0a when 0b review surfaces type-design issues |

## 13. Self-review audit trail

Seven rounds of self-review were run against this design. Issue counts per round: 14 → 9 → 5 → 4 → 2 → 2 → 0. Notable structural changes made during review:

- Foundation PR split into 0a (`bin-common-handler`) and 0b (`bin-api-manager`) to decouple dependency ordering.
- RPC carrier contract made concrete via `sock.Response.DataType = "voipbin_error"` (additive, non-breaking).
- Sentinel-vs-typed-construction guidance added (preference for typed construction with context).
- `request_id` propagation to downstream RPC via `sock.Request.RequestID`.
- Existing `ratelimit.go` + `authenticate.go` middleware added to PR 0b scope (they emit HTTP errors too).
- Translator includes `defer recover()` for panic safety.
- `context.Canceled` / `context.DeadlineExceeded` explicitly handled.
- `details` field reserved in OpenAPI schema for forward compatibility.
- Reason-code catalog RST page added.
- api-validator companion PRs scoped per migration PR.
- Frontend-client audit added as blocking pre-flight for PR 1.
