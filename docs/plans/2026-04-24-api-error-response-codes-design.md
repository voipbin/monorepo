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
    Details []map[string]any `json:"details,omitempty"` // reserved for future per-field detail
    Cause   error  `json:"-"`        // server-side only, never serialized
}

func (e *VoipbinError) Error() string
func (e *VoipbinError) Unwrap() error           // supports errors.Is / errors.As
func (e *VoipbinError) Wrap(cause error) *VoipbinError

// Constructors — one per canonical status. The `domain` parameter
// is typed as outline.ServiceName for compile-time rejection of
// service-name typos; the struct field stays string on the wire.
func InvalidArgument(domain outline.ServiceName, reason, message string) *VoipbinError
func Unauthenticated(domain outline.ServiceName, reason, message string) *VoipbinError
func PaymentRequired(domain outline.ServiceName, reason, message string) *VoipbinError
func PermissionDenied(domain outline.ServiceName, reason, message string) *VoipbinError
func NotFound(domain outline.ServiceName, reason, message string) *VoipbinError
func AlreadyExists(domain outline.ServiceName, reason, message string) *VoipbinError
func FailedPrecondition(domain outline.ServiceName, reason, message string) *VoipbinError
func ResourceExhausted(domain outline.ServiceName, reason, message string) *VoipbinError
func Unavailable(domain outline.ServiceName, reason, message string) *VoipbinError
func Internal(domain outline.ServiceName, reason, message string) *VoipbinError
```

**Domain naming:** uses `commonoutline.ServiceName*` constants (`"call-manager"`, `"billing-manager"`, `"api-manager"`, etc.).

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

### 6.1 Default error-block per endpoint class (OpenAPI wiring)

Each OpenAPI path's `responses:` block MUST reference the named error responses below as a baseline. Endpoints with special semantics (billing-sensitive → 402, conflict-able state transitions → 409, rate-limited paths → 429) add the corresponding response on top of the baseline. Deviations (see counter-examples at the end of this section) MUST be explicitly justified in the PR description.

| Endpoint class | Baseline error responses |
|---|---|
| Read (GET, no path param) | `401`, `500` |
| Read (GET with resource ID) | `400`, `401`, `403`, `404`, `500` |
| Write (POST / PUT / PATCH / DELETE, no resource ID) | `400`, `401`, `500` |
| Write (POST / PUT / PATCH / DELETE with resource ID) | `400`, `401`, `403`, `404`, `500` |
| Admin-gated (hasPermission check, any method, no resource ID) | baseline + `403` |
| Billing-sensitive (success path deducts credits) | baseline + `402` |
| Rate-limited | baseline + `429` |
| State-transition (operation invalid for current resource state) | baseline + `409` |
| RPC-heavy (success path fans out to ≥2 internal managers) | baseline + `503` |

Rationale:
- `500` is always possible — translator default or panic recovery.
- `401` is always possible on authenticated endpoints.
- `400` is possible on writes that parse a body AND on any endpoint parsing a path-param UUID (malformed or zero-value UUID → INVALID_ID).
- `403` and `404` apply when there's a resource to authorize or find, or when the endpoint is admin-gated (even collection endpoints — `hasPermission(PermissionProjectSuperAdmin)` on a collection emits 403 for non-admins).
- `402` / `409` / `429` are opt-in per endpoint semantics.

Billing-sensitive = any endpoint whose success path deducts credits from the customer's balance (e.g., `POST /calls`, `POST /messages`, `POST /emails`, `POST /numbers`). The 402 response is used for insufficient balance.

**State-transition rationale.** Many resources have state machines: a call goes `dialing → ringing → progressing → hangup`; a recording goes `inactive → active`; a transfer requires the source call to be `progressing`. An operation that's invalid for the current state (e.g., `POST /calls/{id}/hangup` on an already-hung-up call, `POST /calls/{id}/recording-start` on an already-recording call) returns `409 FAILED_PRECONDITION`. Use a domain-specific reason code (`CALL_ALREADY_HANGUP`, `RECORDING_ALREADY_ACTIVE`, `CALL_STATE_INVALID`) so clients can branch precisely.

**RPC-heavy rationale.** Endpoints that fan out to multiple internal managers in their success path can fail mid-fan-out from any of them. `POST /calls`, for example, calls call-manager + flow-manager + number-manager + billing-manager. A transient RabbitMQ failure or downstream manager outage surfaces as `503 UNAVAILABLE`. Single-manager endpoints (read paths in particular) typically don't add `503` to their baseline because their failure surface is dominated by `INTERNAL` (single hop).

Counter-examples (endpoints that deliberately deviate — each MUST cite this list in its OpenAPI spec):
- `/ping` — unauthenticated health check. Only `500` applies.
- Auth stubs (`POST /v1.0/auth/boot` etc.) — entire purpose is `404 ROUTE_NOT_FOUND`; baseline does not apply.
- Enumeration-safe endpoints (`POST /auth/signup`, `POST /auth/email-verify`, `POST /auth/unregister`) — deliberately return `200` on both success and failure to prevent email/account enumeration. Their specs declare minimal error responses on parse/shape errors only (`400`), not on logical failure.
- WebSocket upgrade endpoints (`/ws`, `/service-agents/ws`) — handshake errors use HTTP (and go through this baseline), but post-upgrade errors use WebSocket close codes out of HTTP scope.

Applied starting PR 1b — see `docs/plans/2026-04-25-api-error-pr1b-admin-agent-ui-plan.md` for the first full application. PR 0b's `/me` and PR 1's `/customer*` pre-date this convention and will be retrofit in a single sweep alongside §10.5's "shrink fallback" PR.

Known gap: HTTP 405 Method Not Allowed has no canonical status equivalent in the enum. `server/no_route.go` documents that a wrong-method request currently falls through to `ROUTE_NOT_FOUND` / 404. A future coordinated schema update to add `METHOD_NOT_ALLOWED` will close this gap; until then, migration PRs do not need to declare 405 on their paths.

## 7. `bin-api-manager` server-layer changes

### 7.1 Helpers (`server/error.go` and `server/error_test.go`)

```go
// server/error.go (production code)
func abortWithError(c *gin.Context, err *cerrors.VoipbinError)
func abortWithServiceError(c *gin.Context, err error)   // runs translator

// server/error_test.go (test helper — kept out of the production binary
// per Go convention so `testing` is not imported by runtime code)
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder,
                          status cerrors.Status, reason string,
                          domain commonoutline.ServiceName)
```

The Status-to-HTTP mapping lives in `bin-common-handler/models/errors` as the exported `HTTPStatusFor(cerrors.Status) int` (10-entry switch) so it can be reused by both the RPC carrier (`ToResponse`) and this abort helper without duplication.

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
3. **Transport failure detection:** `context.Canceled` → maps to `UNAVAILABLE` with reason `REQUEST_CANCELED` (full envelope returned; the client is likely gone but the envelope is built anyway so server-side logging still captures the correlation ID). `context.DeadlineExceeded` → maps to `UNAVAILABLE` with reason `REQUEST_TIMEOUT`. RabbitMQ transport errors fall through to the substring fallback (`"unavailable"`) → `Unavailable("api-manager", "SERVICE_UNAVAILABLE", ...)`.
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
- Exported `HTTPStatusFor(Status) int` — single source of truth for the canonical-status → HTTP-status mapping, consumed by both `ToResponse` (RPC) and bin-api-manager's abort helper (HTTP).
- Must land before any api-manager work depends on it.

### 10.2 PR 0b — `bin-api-manager` infrastructure

- `abortWithError` / `abortWithServiceError` / `assertErrorResponse` helpers + tests (reuses `cerrors.HTTPStatusFor` from bin-common-handler).
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

### Round 1 implementation refinements (post-design, pre-merge)

During PR #799 self-review, six refinement commits tightened the API before merge:

- Exported `HTTPStatusFor` (was private `httpStatusFor`) so bin-api-manager can reuse the mapping without duplication.
- `Wrap()` now returns a shallow copy instead of mutating the receiver, eliminating aliasing surprises when error pointers are memoized or passed across goroutines.
- Added reserved `Details []map[string]any` field with `omitempty` so per-field validation detail can land later without a breaking schema change.
- Typed the constructor `domain` parameter as `outline.ServiceName` to catch service-name typos at compile time.
- Expanded `ToResponse` round-trip tests to assert all four wire fields; added per-status `FromResponse` round-trip assertion.
- Documentation: package godoc states the cerrors import alias convention and admission-rule justification; `DataTypeVoipbinError` documents additive-only wire-format contract; Error() documented as non-client-safe; sock.Request.RequestID doc points to the design's propagation chain.

### Round 2 refinements

- `Wrap()` doc comment extended to acknowledge the shallow-copy aliasing semantics of the Details slice backing array.
- `TestHTTPStatusFor` now covers the empty-string Status case (zero-value safety).
- Removed the redundant `outline` import alias across `constructors.go`, `constructors_test.go`, and `rpc_test.go`.
- `FromResponse` godoc corrected to enumerate all four nil-return guards (nil response / success status / wrong DataType / empty Data / unmarshal failure).

### Round 3 refinements

- Design doc §10.2 updated to no longer reference the private `httpStatusFor`; PR 0b's `abortWithError` reuses the exported `cerrors.HTTPStatusFor`.
- `TestFromResponseEmptyData` added to cover the DataType-matches-but-Data-empty branch with both `nil` and empty-slice inputs.

### PR 1 + PR 1b refinements (2026-04-24 / 2026-04-25)

**PR 1 (customer-facing auth & identity):**
- Migrated 5 auth stub handlers and `/v1.0/customer*` (5 handlers, 16 sites) to the canonical envelope.
- Added translator substring patterns for `"direct access not supported"` and `"does not belong to this customer"` → PERMISSION_DENIED (fixes 500→403 regression that would have cascaded to every servicehandler using these strings).
- Added `SERVICE_UNAVAILABLE` to the RST catalog.
- Documented `INVALID_JSON_BODY` and `INVALID_ID` reasons.
- Established test helper `assertMissingAuthIdentity` for per-handler missing-auth coverage.

**PR 1b (admin `/customers*` + agent-UI `/service-agents/*`):**
- Migrated admin `/customers` (10 handlers, 35 sites) and agent-UI `/service-agents/me*` / `/service-agents/customer` (6 handlers, 16 sites).
- Codified the default error-block convention (§6.1) — read-no-ID vs read-with-ID vs write-no-ID vs write-with-ID.
- Applied the convention to 12 OpenAPI path files.
- `/me` (PR 0b) and `/customer*` (PR 1) remain on their original simpler error-block declarations; retrofit deferred to next modification of those paths.

### PR 2 calls-group refinements (2026-04-25)

- §6.1 extended with two new endpoint classes: state-transition (baseline + 409) and RPC-heavy (baseline + 503). Concrete billing-sensitive examples enumerated.
- 92 error sites migrated across 27 handlers in 5 files (`calls.go`, `groupcalls.go`, `recordings.go`, `recordingfiles.go`, `transfers.go`).
- New `call-manager` domain section in the RST reason-code catalog, populated with `CALL_NOT_FOUND`, `CALL_ALREADY_HANGUP`, `CALL_STATE_INVALID`, `RECORDING_NOT_FOUND`, `RECORDING_ALREADY_ACTIVE`, `RECORDING_NOT_ACTIVE`, `INSUFFICIENT_BALANCE`, `GROUPCALL_NOT_FOUND`.
- WebSocket counter-example confirmed in practice: `GET /calls/{id}/media-stream` declares only handshake-level error responses; post-upgrade errors stay out of HTTP scope.
