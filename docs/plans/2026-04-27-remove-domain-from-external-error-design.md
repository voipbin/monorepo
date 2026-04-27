# Design: Remove `domain` from external HTTP error responses

**Date:** 2026-04-27
**Branch:** `NOJIRA-remove-domain-from-external-error`
**Status:** Draft (brainstorming → design)

---

## 1. Problem

The canonical error envelope returned by `bin-api-manager` to external API clients
currently includes a `domain` field carrying the **internal** microservice name that
originated the error (e.g., `"call-manager"`, `"billing-manager"`, `"api-manager"`).

Example today (404 from a call lookup):

```json
{
  "error": {
    "status": "NOT_FOUND",
    "reason": "CALL_NOT_FOUND",
    "domain": "call-manager",
    "message": "The call was not found.",
    "request_id": "9f3d4c2a-7e1b-4f6e-9c5a-2b1d8e7f3a0c"
  }
}
```

External clients have no way to interpret these internal service names — they reflect
our backend topology, not the customer-facing product. Exposing them:

- Leaks internal architecture (what services exist and which surface which errors).
- Adds a field that external consumers cannot meaningfully act on (`reason` is the
  actionable identifier).
- Couples the public contract to internal refactors (renaming or splitting a manager
  would observably change the error envelope).

The same `domain` value remains genuinely useful **inside** the system: across RPC
between managers (so a downstream service knows which upstream emitted the typed
error) and in server-side logs.

## 2. Goal

Stop emitting `domain` in external HTTP error JSON bodies. Keep it everywhere
internal: the `VoipbinError` Go struct, the RPC `ToResponse`/`FromResponse` path,
and the server-side `Error()` log format are unchanged.

### 2.1 Non-goals

- Removing the `Domain` field from the `VoipbinError` Go struct.
- Changing internal RPC error propagation between managers.
- Changing the `Error()` log format.
- Renaming reason codes (e.g., `CALL_NOT_FOUND`, `INSUFFICIENT_BALANCE`) — they stay
  unique and self-describing.
- Changing webhook payloads or WebSocket event shapes (they do not carry
  `VoipbinError`).
- Backward compatibility for any external client currently branching on
  `error.domain` — accepted as a clean breaking change. `reason` is the documented
  branch field and remains stable.

## 3. Scope

### 3.1 In scope

| Surface | Today | After |
|---|---|---|
| `VoipbinError` Go struct | has `Domain` | unchanged |
| Internal RPC (`bin-common-handler/models/errors/rpc.go`) | carries `Domain` over RabbitMQ | unchanged |
| `Error()` method | prints `<domain>: <reason>: <message>` | unchanged |
| **External HTTP error JSON body** | includes `"domain"` | **`domain` key omitted** |
| Public RST docs (`bin-api-manager/docsdev/source/`) | document `domain` | drop references; rebuild HTML |
| OpenAPI error schema (`bin-openapi-manager/openapi/openapi.yaml`) | declares `domain` | remove field; regenerate types |

### 3.2 Out of scope

- Webhook delivery (`bin-webhook-manager`) — payloads are per-resource
  `WebhookMessage` business types, not `VoipbinError`. No change.
- WebSocket event push (`bin-api-manager/pkg/websockhandler`) — same: business
  events, not error envelopes. No change.
- Other manager services — they continue to construct `cerrors.X(serviceName, ...)`
  unchanged. The boundary that strips `domain` lives only in the api-manager HTTP
  response writer.
- **`gin.Recovery()` panic-recovery path** — `cmd/api-manager/main.go` uses
  `gin.Default()`, which installs `gin.Recovery()`. A panic inside any handler
  is caught and answered with `c.AbortWithStatus(500)` — empty body, no
  envelope, **no `domain` field** (no leak today, no leak after this change).
  The fact that the recovery path returns an empty body instead of the
  canonical envelope is a pre-existing gap with the contract claim in
  `restful_api_errors.rst`, but fixing it requires replacing `gin.Default()`
  with `gin.New()` plus a custom `RecoveryWithWriter` that calls
  `apierror.EnvelopeFor(nil, requestID)` — a separate concern unrelated to
  the `domain`-removal goal. **Tracked as a follow-up**, not addressed here.
  The integration test in §6.2.1 includes a panic case as a *current-behavior
  assertion* (empty body, 500 status, no leak) so a future fix surfaces as a
  test diff.
- **`bin-api-manager/lib/service/` `c.AbortWithStatus(400)` family** —
  the auth-related handlers (`auth.go`, `signup.go`, `boot.go`,
  `unregister.go`, plus password-forgot/reset, email-verify variants
  registered in `cmd/api-manager/main.go`) emit `c.AbortWithStatus(400)`
  on validation failure. This produces an empty body — no envelope, no
  `domain` field (no leak today, no leak after this change). Same
  disposition as the `gin.Recovery` gap: a pre-existing contract gap with
  `restful_api_errors.rst` ("every 4xx/5xx response... contains an
  `error.reason` field"), and a separate concern unrelated to the
  `domain`-removal goal. **Tracked as a follow-up**: convert these sites
  to construct `cerrors.InvalidArgument(...)` and route through
  `apierror.EnvelopeFor`. The §6.2.2 grep guard includes `lib/service/` in
  its scope so any future engineer who *does* convert one of these sites
  cannot accidentally re-introduce the `domain` key. The §6.5 smoke check
  includes one `lib/service/` endpoint as a *current-behavior assertion*
  (empty 400 body) for the same reason as the panic case.

## 4. Approach

A small new package, `bin-api-manager/lib/apierror/`, owns the external HTTP error
envelope shape. A single function — `EnvelopeFor` — converts a `*VoipbinError` (plus
the request ID) into the `gin.H` body. Three existing call sites that today
open-code the envelope are refactored to call this function.

### 4.1 Why a new package?

Today the envelope is built in **four** places:

1. `bin-api-manager/server/error.go` — `abortWithError` (the main path; routes
   typed `*VoipbinError` from servicehandlers and the `NoRoute` handler).
2. `bin-api-manager/lib/middleware/authenticate.go` — `abortUnauthenticated`
   (401, raw status/reason/message).
3. `bin-api-manager/lib/middleware/authenticate.go` — `isFrozenAccountBlocked`
   inline `c.AbortWithStatusJSON` (403 `ACCOUNT_FROZEN`, includes a `details`
   payload with deletion timestamps and recovery endpoint). **Easily missed**
   because it does not go through a small named helper.
4. `bin-api-manager/lib/middleware/ratelimit.go` — rate-limit branch (429
   `RATE_LIMIT_EXCEEDED`).

All four sites today emit `"domain": string(commonoutline.ServiceNameAPIManager)`
or `"domain": e.Domain`. After this change all four go through a single helper
that omits the field.

The middleware files cannot import `server/` (would create an import cycle),
so the helper must live in a third package both can depend on. The middleware
files explicitly note this constraint in their existing comments. A small
`lib/apierror/` package is the cleanest fit and avoids forcing a `gin` dependency
into `bin-common-handler`.

### 4.2 The helper

```go
// Package apierror builds the external HTTP error envelope for the
// VoIPbin public API. The envelope intentionally omits the internal
// Domain (originating service name) carried by VoipbinError — that field
// is internal-only and must not cross the public API boundary.
package apierror

import (
    "github.com/gin-gonic/gin"

    cerrors "monorepo/bin-common-handler/models/errors"
    commonoutline "monorepo/bin-common-handler/models/outline"
)

// EnvelopeFor returns the JSON body for an external HTTP error response.
// Pass the request ID extracted from the gin context. A nil VoipbinError
// falls back to a generic INTERNAL envelope so callers never panic on a
// missed nil check.
func EnvelopeFor(e *cerrors.VoipbinError, requestID string) gin.H {
    if e == nil {
        e = cerrors.Internal(
            commonoutline.ServiceNameAPIManager,
            "INTERNAL",
            "An internal error occurred.",
        )
    }
    body := gin.H{
        "status":     string(e.Status),
        "reason":     e.Reason,
        "message":    e.Message,
        "request_id": requestID,
    }
    if len(e.Details) > 0 {
        body["details"] = e.Details
    }
    return gin.H{"error": body}
}
```

Key properties:

- **Single chokepoint.** Future internal-only fields added to `VoipbinError` do
  not leak unless a developer explicitly adds them here.
- **No struct change.** `VoipbinError.Domain` stays. RPC, logs, and constructors
  are untouched.
- **Nil-safe.** Mirrors the existing `abortWithError` nil guard so behavior at the
  edge is unchanged for that case.
- **Preserves `details`.** Conditional inclusion matches today's behavior.

### 4.3 Refactor of the four call sites

| Site | Before | After |
|---|---|---|
| `server/error.go` `abortWithError` | open-coded `gin.H` with `"domain"` | `c.AbortWithStatusJSON(cerrors.HTTPStatusFor(e.Status), apierror.EnvelopeFor(e, middleware.RequestIDFromContext(c)))` |
| `lib/middleware/authenticate.go` `abortUnauthenticated` | open-coded `gin.H` from raw status/reason/message | **Keep the existing public signature** `abortUnauthenticated(c *gin.Context, reason, message string)` so the two callers (`authenticate.go:37` and `:56`) need no diff. Refactor only the body to construct `cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, reason, message)` and route through `apierror.EnvelopeFor(e, RequestIDFromContext(c))`. |
| `lib/middleware/authenticate.go` `isFrozenAccountBlocked` | inline open-coded `gin.H` with `details` | construct via `cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "ACCOUNT_FROZEN", "This account is frozen. Contact support.")`, attach the existing `details` payload via the `Details []map[string]any` field on `VoipbinError`, then call `apierror.EnvelopeFor` |
| `lib/middleware/ratelimit.go` rate-limit branch | open-coded `gin.H` | construct via `cerrors.ResourceExhausted(commonoutline.ServiceNameAPIManager, "RATE_LIMIT_EXCEEDED", "...")`, then call `apierror.EnvelopeFor` |

The middleware sites also gain consistency: they now go through the same typed
constructors as the rest of the codebase instead of hand-assembling status
strings. The `isFrozenAccountBlocked` site fits naturally because `VoipbinError`
already has a `Details []map[string]any` field that the helper preserves
verbatim.

**Verified prerequisites for the `isFrozenAccountBlocked` refactor**
(`bin-common-handler/models/errors/`):

- `PermissionDenied(domain outline.ServiceName, reason, message string) *VoipbinError`
  constructor exists in `constructors.go` with the required signature — no new
  `bin-common-handler` change needed.
- `VoipbinError.Details []map[string]any` is exported and assignable post-
  construction (`voipbin_error.go:26`). The pattern is:
  ```go
  e := cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager,
      "ACCOUNT_FROZEN", "This account is frozen. Contact support.")
  e.Details = details
  c.AbortWithStatusJSON(cerrors.HTTPStatusFor(e.Status),
      apierror.EnvelopeFor(e, middleware.RequestIDFromContext(c)))
  ```
- No `bin-common-handler` API change required, so the verification scope in
  §6.4 stays bounded to `bin-api-manager` + `bin-openapi-manager`.

### 4.4 Defense-in-depth: enforcement beyond convention

The `EnvelopeFor` chokepoint is a *convention*. Nothing in the type system
prevents a future caller from doing `c.JSON(status, voipbinErr)` directly,
which would re-leak `Domain` because the struct tag is `json:"domain"`
(no `omitempty`, not `json:"-"`).

We deliberately do **not** change the JSON tag to `json:"-"`. The internal
RPC path (`bin-common-handler/models/errors/rpc.go` `ToResponse`/`FromResponse`)
relies on `json.Marshal(e)` producing `Domain` so that managers downstream
of an RPC failure see which upstream service emitted the typed error. Stripping
`Domain` from the JSON tag would break this without an equivalent side-channel,
which is a much larger blast-radius change than the leak we are fixing.

Instead, we add two structural enforcements:

1. **Test-level negative regression.** Every test case in `envelope_test.go`
   asserts the absence of the `domain` key (see §6.1). This catches any future
   change to `EnvelopeFor` that re-adds the field.

2. **CI-level grep guard** added to the verification workflow (or the existing
   `scripts/check-docs.sh`-style hook). One pattern flagged:
   - Any `"domain"\s*:` literal under `bin-api-manager/server/` or
     `bin-api-manager/lib/middleware/` (excluding `*_test.go` files and the
     new `lib/apierror/` package itself). Catches new open-coded envelopes
     that re-introduce the field. This is a high-signal, low-false-positive
     check: the literal string `"domain":` has no other plausible reason to
     appear in those directories.

   The implementation plan will determine whether to add this to the existing
   `Makefile` lint target, the `.claude/scripts/check-docs-size.sh` hook
   framework, or a new `scripts/check-error-envelope.sh`.

3. **Direct `*VoipbinError` serialization is NOT caught by a regex.** The
   realistic regression — `c.JSON(status, e)` where `e` is a typed
   `*cerrors.VoipbinError` local — does not contain the literal token
   `VoipbinError` at the call site, so any regex on the call expression
   would either miss the case or flood with false positives. The structural
   defenses against this case are:
   - The integration test in §6.2.1 (drives the full handler chain and
     asserts on the actual response body, catching any code path that
     bypasses `EnvelopeFor`).
   - The negative regression unit test in §6.1.
   - A future enhancement (out of scope for this design): a custom
     `go/analysis` analyzer or `ruleguard` rule that flags any gin response
     method (`c.JSON | c.AbortWithStatusJSON | c.IndentedJSON | c.SecureJSON
     | c.JSONP`) whose 2nd argument's static type is `*cerrors.VoipbinError`.
     A type-aware analyzer is the only reliable way to catch this; it should
     be tracked as a follow-up if the team wants belt-and-suspenders
     enforcement.

### 4.5 Resulting external response shape

Same scenarios as today, with `domain` omitted:

**404 from call-manager:**

```json
{
  "error": {
    "status": "NOT_FOUND",
    "reason": "CALL_NOT_FOUND",
    "message": "The call was not found.",
    "request_id": "9f3d4c2a-7e1b-4f6e-9c5a-2b1d8e7f3a0c"
  }
}
```

**401 from auth middleware:**

```json
{
  "error": {
    "status": "UNAUTHENTICATED",
    "reason": "AUTHENTICATION_REQUIRED",
    "message": "Authentication is required.",
    "request_id": "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d"
  }
}
```

**402 from billing-manager (insufficient balance):**

```json
{
  "error": {
    "status": "PAYMENT_REQUIRED",
    "reason": "INSUFFICIENT_BALANCE",
    "message": "Insufficient account balance.",
    "request_id": "7c8d9e0f-1a2b-3c4d-5e6f-7a8b9c0d1e2f"
  }
}
```

**400 with structured `details`:**

```json
{
  "error": {
    "status": "INVALID_ARGUMENT",
    "reason": "INVALID_FIELD",
    "message": "Validation failed.",
    "request_id": "abc...",
    "details": [
      {"field": "destination", "issue": "must be E.164"}
    ]
  }
}
```

## 5. Documentation changes

### 5.1 RST (`bin-api-manager/docsdev/source/`)

1. **`restful_api.rst`** — drop "the originating `domain`" from the envelope
   description (~line 108) and remove the `domain` row from the field table
   further down the page.
2. **`restful_api_errors.rst`** — restructure the reason catalogue with two
   group types:

   **a) Generic / Cross-cutting Reasons** (listed first — applies to every
   endpoint regardless of resource):

   | Reason | Today's home in the file |
   |---|---|
   | `INTERNAL` | api-manager section |
   | `INVALID_ARGUMENT` | api-manager section |
   | `INVALID_JSON_BODY` | api-manager section |
   | `INVALID_ID` | api-manager section |
   | `REQUEST_TIMEOUT` | api-manager section |
   | `REQUEST_CANCELED` | api-manager section |
   | `SERVICE_UNAVAILABLE` | api-manager section |
   | `RESOURCE_NOT_FOUND` | api-manager section |
   | `STATE_INVALID` | api-manager section |
   | `INSUFFICIENT_BALANCE` | api-manager section (translator-mapped) |
   | `RATE_LIMIT_EXCEEDED` | api-manager section |
   | `ACCOUNT_FROZEN` | api-manager section |
   | `PERMISSION_DENIED` | api-manager section |
   | `DIRECT_ACCESS_NOT_SUPPORTED` | api-manager section |
   | `AUTHENTICATION_REQUIRED` | api-manager section |
   | `ROUTE_NOT_FOUND` | api-manager / route-manager section |

   None of these have a clean resource-prefix; they apply across endpoints.
   Grouping them as "Generic" matches how a client should mentally model them
   (try-catch-everywhere) and avoids forcing them into a misleading bucket.

   **b) Resource-Prefixed Reasons** — group by the prefix that already
   appears in the reason code itself (intrinsic to the public contract,
   stable across internal refactors):

   - "Call Reasons": `CALL_*`
   - "Flow Reasons": `FLOW_*`, `ACTIVEFLOW_*`
   - "Recording Reasons": `RECORDING_*`
   - "Number Reasons": `NUMBER_*`, `IDENTITY_VERIFICATION_REQUIRED` (the
     latter is number-flow specific)
   - "Trunk Reasons": `TRUNK_*`
   - "Provider Reasons": `PROVIDER_*`, `PROVIDERCALL_*`
   - "Customer Reasons": `CUSTOMER_*`, `ACCESSKEY_*`
   - …and so on for each prefix actually present in the file. The
     implementation plan will enumerate the exact prefix list after a final
     pass over the source.

   Notes on what to remove:

   - The introductory note that explains how `domain` surfaces via the
     typed-passthrough translator (no longer surfaces externally).
   - All "emitted today by X-manager (via `cerrors.NotFound("<service>", ...)`)"
     provenance prose. That detail is now internal-only by design and risks
     drifting when services are refactored.

   Notes on what stays:

   - Reason codes themselves (e.g., `CALL_NOT_FOUND`, `INSUFFICIENT_BALANCE`,
     `ROUTE_NOT_FOUND`) are **unchanged** and remain 1:1 with what the API
     emits today.

   Do **not** group by "functional area" with prose like "These are emitted
   by billing-manager" — a subset of reasons (notably `INSUFFICIENT_BALANCE`,
   `STATE_INVALID`, `RESOURCE_NOT_FOUND`, `SERVICE_UNAVAILABLE`) are
   translator-mapped through api-manager rather than emitted directly by the
   implied upstream service.
3. **Rebuild and force-add** built HTML, per the root CLAUDE.md RST sync rule:
   ```bash
   cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
   git add -f bin-api-manager/docsdev/build/
   ```

### 5.2 OpenAPI (`bin-openapi-manager/openapi/openapi.yaml`)

1. Locate the error envelope schema referenced by all 4xx/5xx responses.
2. Remove the `domain` property and its description.
3. Regenerate:
   ```bash
   cd bin-openapi-manager && go generate ./...
   cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./...
   ```

## 6. Testing

### 6.1 New unit tests — `bin-api-manager/lib/apierror/envelope_test.go`

Table-driven tests for `EnvelopeFor`, covering:

- Full `VoipbinError` (status/reason/domain/message all set) → resulting `gin.H`
  contains `status`, `reason`, `message`, `request_id`; `details` absent;
  **no `domain` key**.
- `VoipbinError` with non-empty `Details` → `details` key included.
- `VoipbinError` with nil/empty `Details` → `details` key absent.
- `nil` `VoipbinError` → falls back to the internal `INTERNAL` envelope
  (mirrors today's `abortWithError` nil guard).
- `request_id` empty string → still rendered (matches current behavior).
- **Negative regression assertion** (load-bearing): for every case, assert
  `_, ok := body["error"].(gin.H)["domain"]; ok == false`. Catches any future
  re-leak of the field.

### 6.2 Existing tests to update

- `bin-api-manager/server/error_test.go` — drop `domain` from JSON body
  assertions in `abortWithError` / `abortWithServiceError` tests.
- `bin-api-manager/server/error_translate_test.go` — if it asserts the full
  body shape end-to-end, drop the `domain` assertion. The `Domain` on the
  *typed error* still flows internally so translator behavior is unchanged.
- `bin-api-manager/server/no_route_test.go` — if it asserts the response
  body for `ROUTE_NOT_FOUND`, drop the `domain` assertion.
- Middleware tests for `authenticate.go` (`abortUnauthenticated` AND
  `isFrozenAccountBlocked`) and `ratelimit.go` — drop `domain` assertions.
- Sweep: `grep -rn '"domain"' bin-api-manager --include="*_test.go"` to find
  any other assertion that will need updating.

### 6.2.1 New end-to-end integration test

Add an integration test that drives a full gin handler chain end-to-end and
asserts on the actual HTTP response body, not on `EnvelopeFor` in isolation.
This catches regressions where a future caller bypasses the helper entirely
(e.g., a `defer recover()` panic-handler that writes its own envelope, or a
new middleware that does not go through the helper).

Coverage:
- Typed error path: a stub handler that returns `cerrors.NotFound(...)` →
  full request → assert no `domain` in response body.
- Nil-error fallback: invoke `abortWithError(c, nil)` → assert generic
  `INTERNAL` envelope with no `domain`.
- 401 from `abortUnauthenticated` middleware.
- 403 from `isFrozenAccountBlocked` middleware (with `details` payload
  preserved verbatim and `details` key still present in body).
- 429 from `ratelimit.go`.
- 404 from `NoRoute` handler (`ROUTE_NOT_FOUND`).
- **Panic-recovery current-behavior assertion**: a stub handler that calls
  `panic("boom")` → assert HTTP 500, empty body, no `domain` (today the
  `gin.Recovery()` middleware emits a bare `c.AbortWithStatus(500)`). This
  is a guardrail against the recovery middleware ever silently switching
  to an envelope shape without going through `EnvelopeFor`. If the team
  later adopts the follow-up in §3.2 (custom recovery middleware that emits
  the canonical envelope), this test becomes the trigger for updating
  expectations to assert the envelope shape with no `domain`.

### 6.2.2 CI grep guard (positive enforcement)

Add a CI step (Makefile target or script invoked by the existing verification
workflow) that exits non-zero on:

```
grep -rEn '"domain"\s*:' \
  bin-api-manager/server/ \
  bin-api-manager/lib/middleware/ \
  bin-api-manager/lib/service/ \
  --include="*.go" --exclude="*_test.go"
```

This is the structural enforcement against new open-coded envelopes that
re-introduce the field. The literal string `"domain":` has no other plausible
reason to appear in those directories. `lib/service/` is included because
the auth-related handlers there are tracked as a follow-up for envelope
adoption (see §3.2); when that follow-up lands, the grep guard catches any
re-leak preemptively.

A second regex was considered to catch direct `*VoipbinError` serialization
(`c.JSON(status, e)` where `e` is a typed local) but rejected as unreliable —
see §4.4 #3 for rationale. The structural defenses for that bypass are the
integration test in §6.2.1 and the negative regression test in §6.1; a custom
`go/analysis` analyzer is recorded there as a follow-up enhancement.

The implementation plan will choose between adding the grep guard to the
`Makefile` lint target, the existing `check-docs-size.sh` hook framework, or
a new `scripts/check-error-envelope.sh`.

### 6.3 No-change tests

- `bin-common-handler/models/errors/*_test.go` — struct unchanged, RPC
  round-trip unchanged. No edits expected.
- Other backend manager services — they continue to construct
  `cerrors.X(serviceName, ...)` unchanged. No test edits expected.

### 6.4 Verification workflow

Per the root CLAUDE.md, run from each service touched:

```bash
# In bin-api-manager (new package + refactored sites + test updates)
go mod tidy && go mod vendor && go generate ./... && \
  go test ./... && golangci-lint run -v --timeout 5m

# In bin-openapi-manager (after dropping domain from error schema)
go generate ./... && go test ./... && \
  golangci-lint run -v --timeout 5m

# RST docs rebuild
cd bin-api-manager/docsdev && rm -rf build && \
  python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```

### 6.5 Manual smoke check (post-deploy to staging)

Hit each of the previously-divergent error paths and confirm none returns
a `domain` key:

```bash
TOKEN="<staging-token>"
BASE="https://api.staging.voipbin.net/v1.0"

# 401 — auth middleware
curl -sS "$BASE/calls" | jq .

# 400 — lib/service/ current-behavior assertion (empty body, no envelope).
# Tracked as a follow-up to convert to canonical envelope; today this MUST
# return an empty body so a future fix surfaces visibly.
curl -sS -o /dev/null -w "%{http_code}\n" \
  -X POST "$BASE/auth/login" -d '{"invalid": "json"' -H "Content-Type: application/json"

# 403 ACCOUNT_FROZEN — auth middleware (requires a frozen-account token)
curl -sS -H "Authorization: Bearer $FROZEN_TOKEN" "$BASE/calls" | jq .

# 404 from a backend manager (typed error via abortWithServiceError)
curl -sS -H "Authorization: Bearer $TOKEN" \
  "$BASE/calls/00000000-0000-0000-0000-000000000000" | jq .

# 404 ROUTE_NOT_FOUND (NoRoute handler)
curl -sS -H "Authorization: Bearer $TOKEN" "$BASE/no-such-endpoint" | jq .

# 429 — ratelimit (loop until limit hit, then assert)
for i in $(seq 1 200); do
  curl -sS -o /dev/null -w "%{http_code}\n" "$BASE/calls"
done | tail -n 5
curl -sS "$BASE/calls" | jq .
```

For every body, confirm: `status`, `reason`, `message`, `request_id` are
present; **no `domain` key**; for 403 the `details` array is preserved.

## 7. Risks and mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| External client branches on `error.domain` and breaks | Low (reason is the documented branch field; domain values are internal names that can't be meaningfully matched) | Document in release notes; the change is small and easy to revert (one helper). |
| RST `restful_api_errors.rst` reframing introduces stale headings or broken cross-refs | Medium | Build the docs locally (`make html` / `sphinx-build`) and grep for `:ref:` targets that reference the renamed sections before commit. |
| OpenAPI schema regen breaks generated types in `bin-api-manager/gens/openapi_server` | Low (regenerated by `go generate`) | The verification workflow runs `go generate ./...` and `go test ./...` — any drift surfaces immediately. |
| A test elsewhere in the monorepo asserts on `error.domain` | Low | Pre-grep before commit: `grep -rn '"domain"' bin-*/...` (excluding vendor/mocks). |
| Future developer re-introduces `domain` by open-coding a new envelope site | Medium (root cause of today's leak — already happened 4 times) | (1) Single-helper refactor unifies all 4 sites. (2) Negative regression test in `envelope_test.go`. (3) CI grep guard (§6.2.2) actively rejects new open-coded `"domain"` keys and direct `*VoipbinError` serialization in `bin-api-manager/server/` and `lib/middleware/`. |
| Future developer bypasses `EnvelopeFor` by `c.JSON(status, voipbinErr)` directly | Medium (struct tag is `json:"domain"`, no `omitempty`) | CI grep guard #2 (§6.2.2) catches direct serialization. Explicit decision NOT to change struct tag (would break internal RPC). Documented in §4.4. |
| Frozen-account 403 path silently keeps leaking `domain` because the inline site is missed | Eliminated | Found during design review; `isFrozenAccountBlocked` is now an explicit refactor target in §4.3. End-to-end integration test (§6.2.1) and smoke check (§6.5) both exercise the path. |

## 8. Rollback

Single concentrated diff, easy to revert:

1. `git revert` the commit.
2. Re-run the verification workflow in `bin-api-manager` to restore vendor and
   generated artifacts.
3. RST docs revert with the commit; rebuild HTML before re-publishing.

No database migrations, no message-shape changes on RabbitMQ, no cross-service
ABI breakage.
