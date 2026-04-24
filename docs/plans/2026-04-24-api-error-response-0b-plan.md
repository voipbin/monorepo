# VoIPbin API Error Response Codes — PR 0b Plan (api-manager infrastructure)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Wire the shared `VoipbinError` vocabulary (from PR 0a) into `bin-api-manager`'s server layer and middleware, add OpenAPI error schemas, and migrate the two middleware sites that already emit HTTP errors today. Zero handler-file migration in this PR — that starts in PR 1.

**Parent:** `docs/plans/2026-04-24-api-error-response-codes-design.md` (design), `docs/plans/2026-04-24-api-error-response-codes-plan.md` (top-level plan).

**Dependency:** PR 0a (`NOJIRA-api-error-response-codes`) must be merged to main OR this branch must remain stacked on top of it until merge. As of plan-write time, PR 0a is `https://github.com/voipbin/monorepo/pull/799` and this branch (`NOJIRA-api-error-response-0b`) is stacked from PR 0a's tip.

**Tech stack:** Go, Gin, `github.com/oklog/ulid/v2` (request-id generation), `github.com/sirupsen/logrus`, `oapi-codegen` (OpenAPI server code generation), Sphinx (RST docs).

---

## Preconditions

- Worktree: `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-0b`, branch `NOJIRA-api-error-response-0b`.
- Stacked on `NOJIRA-api-error-response-codes` (PR 0a). When PR 0a merges, rebase this branch onto `main`.
- `bin-common-handler/models/errors` package exists (delivered in PR 0a): `VoipbinError`, 10 constructors, `FromResponse`, `ToResponse`, `DataTypeVoipbinError`. `sock.Request.RequestID` field exists.
- `ServiceNameAPIManager = "api-manager"` constant exists in `bin-common-handler/models/outline/servicename.go`.

## Files that will be touched (final list)

**New files in `bin-api-manager/`:**
- `pkg/serviceerrors/sentinels.go` (+ test)
- `lib/middleware/request_id.go` (+ test)
- `server/error.go` — `abortWithError`, `abortWithServiceError`, test helper
- `server/error_translate.go` — translator with fallback chain

**Modified files in `bin-api-manager/`:**
- `lib/middleware/authenticate.go` — replace `c.AbortWithStatus(401)` and `c.AbortWithStatusJSON(403, gin.H{...})` with `abortWithError`
- `lib/middleware/ratelimit.go` — replace `c.AbortWithStatusJSON(429, gin.H{...})` with `abortWithError`
- `cmd/api-manager/main.go` — register the request-id middleware at top-level `app.Use(...)`
- `docsdev/source/restful_api.rst` — new envelope + status mapping table
- `docsdev/source/restful_api_errors.rst` — NEW reason-code catalog page (initially empty table)
- `docsdev/source/index.rst` — include the new catalog page in the ToC
- `docsdev/build/` — Sphinx HTML rebuild (force-added)

**New/modified files in `bin-openapi-manager/`:**
- `openapi/openapi.yaml` — add `ErrorBody`, `ErrorResponse`, named responses. Wire error responses into one trivial endpoint for the spike (likely `GET /v1.0/ping` or `GET /v1.0/customer`). Do NOT wire into any other endpoint yet.

## Testing conventions (from PR 0a verification)

- Plain `testing`, no testify.
- White-box test packages (`package errors`, `package middleware`, etc.) — matches sibling files.
- Table-driven tests for enums and mappings; sequential-`if` for struct-field checks.
- `golangci-lint run -v --timeout 5m` must be clean.
- `go test -race` must pass (picked up latent concurrency bugs in middleware).

## Commit hygiene (from PR 0a)

- Every commit title is EXACTLY `NOJIRA-api-error-response-0b` (matches this branch name).
- No AI attribution anywhere.
- Prefer new commits over `--amend` even for small fixes.
- One logical change per commit.

---

## Task 1: Define the sentinel errors

**Files:**
- Create: `bin-api-manager/pkg/serviceerrors/sentinels.go`
- Create: `bin-api-manager/pkg/serviceerrors/sentinels_test.go`

**Why:** The translator (Task 5) uses `errors.Is` to match these sentinels; existing `servicehandler/` sites use `fmt.Errorf("user has no permission")` — those will migrate to sentinels incrementally in PRs 1+.

**Step 1: Write failing test**

```go
// bin-api-manager/pkg/serviceerrors/sentinels_test.go
package serviceerrors

import (
	stderrors "errors"
	"testing"
)

func TestSentinelsExist(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"permission_denied", ErrPermissionDenied, "permission denied"},
		{"not_found", ErrNotFound, "not found"},
		{"authentication_required", ErrAuthenticationRequired, "authentication required"},
		{"direct_access_not_supported", ErrDirectAccessNotSupported, "direct access not supported"},
		{"invalid_argument", ErrInvalidArgument, "invalid argument"},
		{"internal_error", ErrInternal, "internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("sentinel is nil")
			}
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("got %q want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestSentinelsAreDistinct(t *testing.T) {
	if stderrors.Is(ErrPermissionDenied, ErrNotFound) {
		t.Error("ErrPermissionDenied must not match ErrNotFound")
	}
	if stderrors.Is(ErrNotFound, ErrPermissionDenied) {
		t.Error("ErrNotFound must not match ErrPermissionDenied")
	}
}
```

**Step 2: Run failing test**

```bash
cd bin-api-manager && go test ./pkg/serviceerrors/... -v
```

Expected: compile error.

**Step 3: Write minimal implementation**

```go
// bin-api-manager/pkg/serviceerrors/sentinels.go
// Package serviceerrors defines sentinel errors used by servicehandler
// methods in bin-api-manager. The translator in server/error_translate.go
// matches these sentinels (via errors.Is) and converts them into
// VoipbinError responses for the client.
//
// This layer is intentionally thin: servicehandler code can choose to
// construct a richer *cerrors.VoipbinError directly (preferred when the
// site has context like a resource ID), and fall back to a sentinel
// only when there is no specific context to add.
package serviceerrors

import stderrors "errors"

var (
	ErrPermissionDenied         = stderrors.New("permission denied")
	ErrNotFound                 = stderrors.New("not found")
	ErrAuthenticationRequired   = stderrors.New("authentication required")
	ErrDirectAccessNotSupported = stderrors.New("direct access not supported")
	ErrInvalidArgument          = stderrors.New("invalid argument")
	ErrInternal                 = stderrors.New("internal error")
)
```

**Step 4: Run passing test**

```bash
cd bin-api-manager && go test ./pkg/serviceerrors/... -v
```

Expected: PASS.

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-0b
git add bin-api-manager/pkg/serviceerrors/sentinels.go bin-api-manager/pkg/serviceerrors/sentinels_test.go
git commit -m "NOJIRA-api-error-response-0b

Introduce sentinel errors so generic servicehandler errors can be matched via errors.Is in the translator without string matching.

- bin-api-manager: Add pkg/serviceerrors/sentinels.go with 6 sentinels"
```

---

## Task 2: Request-ID middleware

**Files:**
- Create: `bin-api-manager/lib/middleware/request_id.go`
- Create: `bin-api-manager/lib/middleware/request_id_test.go`

**Step 1: Write failing test**

```go
// bin-api-manager/lib/middleware/request_id_test.go
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDGeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	var seen string
	r.GET("/", func(c *gin.Context) {
		seen = RequestIDFromContext(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if seen == "" {
		t.Fatal("RequestIDFromContext returned empty")
	}
	if !strings.HasPrefix(seen, "req_") {
		t.Errorf("id should start with req_ prefix, got %q", seen)
	}
	if got := w.Header().Get("X-Request-Id"); got != seen {
		t.Errorf("response header X-Request-Id = %q want %q", got, seen)
	}
}

func TestRequestIDEchoesInboundHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	var seen string
	r.GET("/", func(c *gin.Context) {
		seen = RequestIDFromContext(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "req_client_supplied")
	r.ServeHTTP(w, req)

	if seen != "req_client_supplied" {
		t.Errorf("did not echo inbound header, got %q", seen)
	}
	if got := w.Header().Get("X-Request-Id"); got != "req_client_supplied" {
		t.Errorf("response header should echo, got %q", got)
	}
}

func TestRequestIDFromContextEmptyWhenAbsent(t *testing.T) {
	ctx := context.Background()
	if got := requestIDFromStdContext(ctx); got != "" {
		t.Errorf("unset context should yield empty, got %q", got)
	}
}
```

**Step 2: Run failing test**

```bash
cd bin-api-manager && go test ./lib/middleware/... -run TestRequestID -v
```

Expected: compile error.

**Step 3: Write minimal implementation**

```go
// bin-api-manager/lib/middleware/request_id.go
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base32"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	headerRequestID  = "X-Request-Id"
	ctxKeyRequestID  = "request_id"
	requestIDPrefix  = "req_"
	requestIDRawBits = 16 // 16 bytes → 26 base32 chars → 30 total w/ prefix
)

// RequestID returns a Gin middleware that ensures every request has a
// unique correlation ID. If the client sends X-Request-Id, that value
// is echoed; otherwise a fresh ULID-like ID is generated. The ID is
// stored in the Gin context, propagated to c.Request.Context(), and
// attached to the X-Request-Id response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(headerRequestID)
		if id == "" {
			id = newRequestID()
		}
		c.Set(ctxKeyRequestID, id)
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), requestIDCtxKey{}, id),
		)
		c.Writer.Header().Set(headerRequestID, id)

		// Make logrus.WithContext pick up the request_id automatically.
		c.Set("logger", logrus.WithField("request_id", id))

		c.Next()
	}
}

// RequestIDFromContext returns the request ID stored in the Gin
// context, or the empty string if RequestID middleware was not run.
func RequestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(ctxKeyRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type requestIDCtxKey struct{}

// requestIDFromStdContext returns the request ID attached to a
// context.Context by this middleware. Used by non-Gin consumers such
// as the reqHandler wrapper that needs to forward the ID into RPC.
func requestIDFromStdContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}

func newRequestID() string {
	buf := make([]byte, requestIDRawBits)
	if _, err := rand.Read(buf); err != nil {
		// Cryptographically impossible in practice; fall back to a
		// fixed sentinel so downstream code never sees an empty ID.
		return requestIDPrefix + "deadbeef"
	}
	return requestIDPrefix + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}
```

**Note:** the plan originally specified ULID (`github.com/oklog/ulid/v2`) — switching to `crypto/rand + base32` avoids an external dependency. The ID format (`req_` + 26 chars) is unchanged. Update the design doc's "30-char Crockford-ULID" wording to "26-char base32" in a later touch-up.

**Step 4: Run passing test**

```bash
cd bin-api-manager && go test ./lib/middleware/... -run TestRequestID -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add bin-api-manager/lib/middleware/request_id.go bin-api-manager/lib/middleware/request_id_test.go
git commit -m "NOJIRA-api-error-response-0b

Add a Gin middleware that tags every request with a correlation ID. Echoes X-Request-Id when present. Propagates into c.Request.Context() and a logrus field for server-side log correlation.

- bin-api-manager: Add lib/middleware/request_id.go"
```

---

## Task 3: Export `HTTPStatusFor` from `bin-common-handler/models/errors`

The PR 0a `httpStatusFor` is private. PR 0b needs it in both `bin-common-handler` and `bin-api-manager`. Rather than duplicate the 10-entry switch, promote to exported.

**This is a PR 0a refinement** — it touches `bin-common-handler`. Because PR 0a is already in review, we have two choices:

**Option A (recommended):** land this as a small follow-up commit on PR 0a before it merges. Ask the reviewer to pick it up as part of the same review.

**Option B:** duplicate the switch in bin-api-manager for PR 0b, accept temporary drift, consolidate in a later cleanup PR.

**Decision:** use Option A. Push the follow-up to PR 0a's branch. If PR 0a has already merged, this task becomes a small first-commit in PR 0b against `bin-common-handler`.

**Files:**
- Modify: `bin-common-handler/models/errors/rpc.go` — rename `httpStatusFor` → `HTTPStatusFor` (capital H).
- Modify: `bin-common-handler/models/errors/rpc_test.go` — update the one call site in `ToResponse` test (indirect — no direct reference needed) and add a direct test for the exported function.

**Step 1: Write failing test (append to rpc_test.go)**

```go
func TestHTTPStatusFor(t *testing.T) {
	tests := []struct {
		status Status
		want   int
	}{
		{StatusInvalidArgument, 400},
		{StatusUnauthenticated, 401},
		{StatusPaymentRequired, 402},
		{StatusPermissionDenied, 403},
		{StatusNotFound, 404},
		{StatusAlreadyExists, 409},
		{StatusFailedPrecondition, 409},
		{StatusResourceExhausted, 429},
		{StatusUnavailable, 503},
		{StatusInternal, 500},
		{Status("UNKNOWN"), 500}, // default
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := HTTPStatusFor(tt.status); got != tt.want {
				t.Errorf("got %d want %d", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run failing test**

Expected: compile error — `HTTPStatusFor` undefined.

**Step 3: Rename in rpc.go**

```go
// BEFORE: func httpStatusFor(s Status) int { ... }
// AFTER:
// HTTPStatusFor maps a canonical Status to an HTTP status code.
// This mapping is the single source of truth for both RPC and HTTP
// layers; bin-api-manager uses it directly.
func HTTPStatusFor(s Status) int { ... }
```

Update the ToResponse body: `httpStatusFor(e.Status)` → `HTTPStatusFor(e.Status)`.

**Step 4: Run all tests**

```bash
cd bin-common-handler && go test ./models/errors/... -v
```

Expected: PASS — existing ToResponse tests continue to pass, new TestHTTPStatusFor passes.

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/rpc.go bin-common-handler/models/errors/rpc_test.go
git commit -m "NOJIRA-api-error-response-0b

Promote httpStatusFor to exported HTTPStatusFor so bin-api-manager can reuse the mapping without duplication. The single source of truth for Status-to-HTTP across RPC and HTTP layers.

- bin-common-handler: Rename httpStatusFor to HTTPStatusFor (exported)"
```

---

## Task 4: Abort helper (`server/error.go`)

**Files:**
- Create: `bin-api-manager/server/error.go`
- Create: `bin-api-manager/server/error_test.go`

**Step 1: Write failing test**

```go
// bin-api-manager/server/error_test.go
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-api-manager/lib/middleware"

	"github.com/gin-gonic/gin"
)

func TestAbortWithErrorSetsStatusAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithError(c, cerrors.NotFound("call-manager", "CALL_NOT_FOUND", "The call was not found."))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d want 404", w.Code)
	}
	var body struct {
		Error struct {
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			Domain    string `json:"domain"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v; body: %s", err, w.Body.String())
	}
	if body.Error.Status != "NOT_FOUND" {
		t.Errorf("wrong status field: %q", body.Error.Status)
	}
	if body.Error.Reason != "CALL_NOT_FOUND" {
		t.Errorf("wrong reason: %q", body.Error.Reason)
	}
	if body.Error.Domain != "call-manager" {
		t.Errorf("wrong domain: %q", body.Error.Domain)
	}
	if body.Error.Message != "The call was not found." {
		t.Errorf("wrong message: %q", body.Error.Message)
	}
	if body.Error.RequestID == "" {
		t.Error("request_id missing from response body")
	}
}

func TestAbortWithServiceErrorHandlesTypedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithServiceError(c, cerrors.PermissionDenied("billing-manager", "BILLING_ACCESS_DENIED", "Not allowed."))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d want 403", w.Code)
	}
}

func TestAssertErrorResponseHelper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithError(c, cerrors.InvalidArgument("api-manager", "INVALID_ID", "bad id"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}
```

**Step 2: Run failing test**

Expected: compile error — `abortWithError`, `abortWithServiceError`, `assertErrorResponse` undefined.

**Step 3: Write minimal implementation**

```go
// bin-api-manager/server/error.go
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gin-gonic/gin"
)

// abortWithError writes the VoipbinError as a JSON body, sets the
// correct HTTP status code, and aborts the Gin context. Request ID is
// read from middleware.RequestIDFromContext so the middleware must run
// before any handler that calls this.
func abortWithError(c *gin.Context, e *cerrors.VoipbinError) {
	if e == nil {
		e = cerrors.Internal("api-manager", "INTERNAL", "An internal error occurred.")
	}
	c.AbortWithStatusJSON(cerrors.HTTPStatusFor(e.Status), gin.H{
		"error": gin.H{
			"status":     string(e.Status),
			"reason":     e.Reason,
			"domain":     e.Domain,
			"message":    e.Message,
			"request_id": middleware.RequestIDFromContext(c),
		},
	})
}

// abortWithServiceError runs any error returned from servicehandler
// through the translator, then aborts with the resulting VoipbinError.
func abortWithServiceError(c *gin.Context, err error) {
	abortWithError(c, translateToVoipbinError(err))
}

// assertErrorResponse is a test helper used across handler tests. It
// asserts the HTTP status code matches the canonical Status and the
// response body's reason field matches.
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, wantStatus cerrors.Status, wantReason string) {
	t.Helper()
	if w.Code != cerrors.HTTPStatusFor(wantStatus) {
		t.Errorf("status code = %d want %d", w.Code, cerrors.HTTPStatusFor(wantStatus))
	}
	var body struct {
		Error struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v; body=%s", err, w.Body.String())
	}
	if body.Error.Status != string(wantStatus) {
		t.Errorf("status field = %q want %q", body.Error.Status, wantStatus)
	}
	if body.Error.Reason != wantReason {
		t.Errorf("reason = %q want %q", body.Error.Reason, wantReason)
	}
}

// keep the unused-http import quiet when assertErrorResponse moves to
// its own file later; delete this once it does.
var _ = http.StatusOK
```

Note: `_ = http.StatusOK` is a quiet keeper. Remove if `http` is used elsewhere.

**Step 4: Run passing test**

```bash
cd bin-api-manager && go test ./server/... -run "TestAbortWith|TestAssert" -v
```

Expected: PASS. These tests also exercise `translateToVoipbinError`, which doesn't exist yet — dependency for Task 5. Temporarily stub: make `translateToVoipbinError` return `cerrors.Internal(...)` by default, so this task's tests pass. Replace in Task 5.

Actually cleaner: implement a throwaway stub at the END of error.go for this task:

```go
// Stubbed — replaced by Task 5.
func translateToVoipbinError(err error) *cerrors.VoipbinError {
	var ve *cerrors.VoipbinError
	if err != nil && errors.As(err, &ve) {
		return ve
	}
	return cerrors.Internal("api-manager", "INTERNAL", "An internal error occurred.")
}
```

**Step 5: Commit**

```bash
git add bin-api-manager/server/error.go bin-api-manager/server/error_test.go
git commit -m "NOJIRA-api-error-response-0b

Add abortWithError and abortWithServiceError helpers that produce the new JSON error envelope with request_id. Includes a test helper (assertErrorResponse) that handler-test files in later PRs will use to cut assertion boilerplate.

- bin-api-manager: Add server/error.go with abort helpers and assertErrorResponse"
```

---

## Task 5: Translator (`server/error_translate.go`)

**Files:**
- Create: `bin-api-manager/server/error_translate.go` (replaces the stub from Task 4)
- Create: `bin-api-manager/server/error_translate_test.go`

**Step 1: Write failing test**

```go
// bin-api-manager/server/error_translate_test.go
package server

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
)

func TestTranslateTypedPassthrough(t *testing.T) {
	in := cerrors.NotFound("call-manager", "CALL_NOT_FOUND", "x")
	out := translateToVoipbinError(in)
	if out != in {
		t.Errorf("typed error should pass through, got %+v", out)
	}
}

func TestTranslateWrappedTypedError(t *testing.T) {
	in := cerrors.NotFound("call-manager", "CALL_NOT_FOUND", "x")
	wrapped := fmt.Errorf("context: %w", in)
	out := translateToVoipbinError(wrapped)
	if out != in {
		t.Errorf("wrapped typed error should unwrap to original, got %+v", out)
	}
}

func TestTranslateSentinels(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus cerrors.Status
		wantReason string
	}{
		{"permission_denied", serviceerrors.ErrPermissionDenied, cerrors.StatusPermissionDenied, "PERMISSION_DENIED"},
		{"not_found", serviceerrors.ErrNotFound, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND"},
		{"auth_required", serviceerrors.ErrAuthenticationRequired, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED"},
		{"direct_access", serviceerrors.ErrDirectAccessNotSupported, cerrors.StatusPermissionDenied, "DIRECT_ACCESS_NOT_SUPPORTED"},
		{"invalid_argument", serviceerrors.ErrInvalidArgument, cerrors.StatusInvalidArgument, "INVALID_ARGUMENT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateToVoipbinError(tt.err)
			if got.Status != tt.wantStatus || got.Reason != tt.wantReason {
				t.Errorf("got status=%q reason=%q want %q/%q", got.Status, got.Reason, tt.wantStatus, tt.wantReason)
			}
		})
	}
}

func TestTranslateTransportErrors(t *testing.T) {
	if out := translateToVoipbinError(context.DeadlineExceeded); out.Status != cerrors.StatusUnavailable {
		t.Errorf("DeadlineExceeded should map to UNAVAILABLE, got %+v", out)
	}
	// context.Canceled maps to UNAVAILABLE with a specific reason.
	if out := translateToVoipbinError(context.Canceled); out.Status != cerrors.StatusUnavailable {
		t.Errorf("Canceled should map to UNAVAILABLE, got %+v", out)
	}
}

func TestTranslateSubstringFallback(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus cerrors.Status
	}{
		{stderrors.New("user has no permission"), cerrors.StatusPermissionDenied},
		{stderrors.New("agent has no permission"), cerrors.StatusPermissionDenied},
		{stderrors.New("agent authentication required"), cerrors.StatusUnauthenticated},
		{stderrors.New("call not found"), cerrors.StatusNotFound},
		{stderrors.New("upstream service unavailable"), cerrors.StatusUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			got := translateToVoipbinError(tt.err)
			if got.Status != tt.wantStatus {
				t.Errorf("got %q want %q", got.Status, tt.wantStatus)
			}
		})
	}
}

func TestTranslateDefault(t *testing.T) {
	orig := stderrors.New("something nobody anticipated")
	got := translateToVoipbinError(orig)
	if got.Status != cerrors.StatusInternal {
		t.Errorf("unknown error should map to INTERNAL, got %q", got.Status)
	}
	if got.Cause != orig {
		t.Errorf("Cause should wrap original error, got %v", got.Cause)
	}
}

func TestTranslateNil(t *testing.T) {
	got := translateToVoipbinError(nil)
	if got == nil {
		t.Fatal("translator must never return nil")
	}
	if got.Status != cerrors.StatusInternal {
		t.Errorf("nil error should map to INTERNAL, got %q", got.Status)
	}
}

func TestTranslatePanicRecovery(t *testing.T) {
	// Translator must not propagate panics — wrap in a type that
	// panics when Error() is called.
	got := translateToVoipbinError(panickingError{})
	if got.Status != cerrors.StatusInternal {
		t.Errorf("panic path must degrade to INTERNAL, got %q", got.Status)
	}
}

type panickingError struct{}

func (panickingError) Error() string { panic("boom") }
```

**Step 2: Run failing test**

Expected: some compile / some fail depending on stub. All must fail or miss until full implementation.

**Step 3: Write implementation** (replaces the stub from Task 4)

```go
// bin-api-manager/server/error_translate.go
package server

import (
	"context"
	stderrors "errors"
	"strings"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-api-manager/pkg/serviceerrors"
)

// translateToVoipbinError maps any error returned from a servicehandler
// into a *VoipbinError. Priority:
//   1. Typed passthrough (errors.As).
//   2. Sentinel match (errors.Is against serviceerrors.Err*).
//   3. Transport-failure detection (context.Canceled/DeadlineExceeded).
//   4. Substring fallback for legacy fmt.Errorf messages (shrinks over time).
//   5. Default: Internal.
// The whole function is wrapped in defer recover() so a panic degrades
// gracefully rather than dropping the response.
func translateToVoipbinError(err error) (out *cerrors.VoipbinError) {
	defer func() {
		if r := recover(); r != nil {
			out = cerrors.Internal("api-manager", "INTERNAL", "An internal error occurred.")
		}
	}()

	if err == nil {
		return cerrors.Internal("api-manager", "INTERNAL", "An internal error occurred.")
	}

	// 1. Typed passthrough.
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		return ve
	}

	// 2. Sentinel match.
	switch {
	case stderrors.Is(err, serviceerrors.ErrAuthenticationRequired):
		return cerrors.Unauthenticated("api-manager", "AUTHENTICATION_REQUIRED", "Authentication is required.")
	case stderrors.Is(err, serviceerrors.ErrPermissionDenied):
		return cerrors.PermissionDenied("api-manager", "PERMISSION_DENIED", "You do not have permission to access this resource.")
	case stderrors.Is(err, serviceerrors.ErrDirectAccessNotSupported):
		return cerrors.PermissionDenied("api-manager", "DIRECT_ACCESS_NOT_SUPPORTED", "Direct access is not supported for this endpoint.")
	case stderrors.Is(err, serviceerrors.ErrNotFound):
		return cerrors.NotFound("api-manager", "RESOURCE_NOT_FOUND", "The requested resource was not found.")
	case stderrors.Is(err, serviceerrors.ErrInvalidArgument):
		return cerrors.InvalidArgument("api-manager", "INVALID_ARGUMENT", "The request is invalid.")
	case stderrors.Is(err, serviceerrors.ErrInternal):
		return cerrors.Internal("api-manager", "INTERNAL", "An internal error occurred.").Wrap(err)
	}

	// 3. Transport failures.
	if stderrors.Is(err, context.Canceled) {
		return cerrors.Unavailable("api-manager", "REQUEST_CANCELED", "The request was canceled.").Wrap(err)
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return cerrors.Unavailable("api-manager", "REQUEST_TIMEOUT", "The request timed out.").Wrap(err)
	}

	// 4. Substring fallback. Intentionally small — each match is a
	//    migration target for sentinels. This switch shrinks over time.
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no permission"):
		return cerrors.PermissionDenied("api-manager", "PERMISSION_DENIED", "You do not have permission to access this resource.").Wrap(err)
	case strings.Contains(msg, "authentication required"):
		return cerrors.Unauthenticated("api-manager", "AUTHENTICATION_REQUIRED", "Authentication is required.").Wrap(err)
	case strings.Contains(msg, "not found"):
		return cerrors.NotFound("api-manager", "RESOURCE_NOT_FOUND", "The requested resource was not found.").Wrap(err)
	case strings.Contains(msg, "unavailable"):
		return cerrors.Unavailable("api-manager", "SERVICE_UNAVAILABLE", "An upstream service is temporarily unavailable.").Wrap(err)
	}

	// 5. Default.
	return cerrors.Internal("api-manager", "INTERNAL", "An internal error occurred.").Wrap(err)
}
```

**Also remove the stub from error.go.**

**Step 4: Run passing test**

```bash
cd bin-api-manager && go test ./server/... -v
```

Expected: every translator test passes.

**Step 5: Commit**

```bash
git add bin-api-manager/server/error.go bin-api-manager/server/error_translate.go bin-api-manager/server/error_translate_test.go
git commit -m "NOJIRA-api-error-response-0b

Replace the translator stub with the full fallback chain: typed passthrough, sentinel match, transport-error detection, substring fallback, INTERNAL default. Wrapped in defer recover so panics degrade gracefully.

- bin-api-manager: Implement translateToVoipbinError with full fallback chain and panic safety"
```

---

## Task 6: Migrate `lib/middleware/ratelimit.go`

**Files:**
- Modify: `bin-api-manager/lib/middleware/ratelimit.go`

Current call at line 74:

```go
c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
    "error":   "rate_limit_exceeded",
    "message": "Too many requests. Please try again later.",
})
```

Replace with:

```go
// Inline — abortWithError lives in server package, import cycle if used here.
// Instead, construct the VoipbinError directly and write it ourselves, using
// the same envelope shape as server.abortWithError.
id := RequestIDFromContext(c)
c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
    "error": gin.H{
        "status":     string(cerrors.StatusResourceExhausted),
        "reason":     "RATE_LIMIT_EXCEEDED",
        "domain":     "api-manager",
        "message":    "Too many requests. Please try again later.",
        "request_id": id,
    },
})
```

**Why not call abortWithError?** `middleware` package cannot depend on `server` package (import cycle). So we inline the envelope construction. Alternative: move `abortWithError` to its own neutral package (e.g., `pkg/httperr`) that both `server` and `middleware` can import. For PR 0b minimal scope, inline. Flag in a follow-up.

**Step 1: Update existing ratelimit test**

The current `ratelimit_test.go` likely asserts on the old envelope. Update its assertions for the new envelope (`body.error.reason == "RATE_LIMIT_EXCEEDED"` etc.).

**Step 2: Run failing test**

Expected: test failure on response body assertion.

**Step 3: Edit ratelimit.go**

Add import for `cerrors`. Replace the abort block.

**Step 4: Run passing test**

Expected: PASS.

**Step 5: Commit**

```bash
git add bin-api-manager/lib/middleware/ratelimit.go bin-api-manager/lib/middleware/ratelimit_test.go
git commit -m "NOJIRA-api-error-response-0b

Emit the new {error: {status, reason, domain, message, request_id}} envelope from the rate-limit middleware so rate-limited clients see the same shape as every other error path.

- bin-api-manager: Migrate lib/middleware/ratelimit.go to the new error envelope"
```

---

## Task 7: Migrate `lib/middleware/authenticate.go`

**Files:**
- Modify: `bin-api-manager/lib/middleware/authenticate.go`

Three sites to migrate (from earlier exploration):
- Line 35: `c.AbortWithStatus(401)` — missing/invalid header.
- Line 54: `c.AbortWithStatus(401)` — auth function failed.
- Line 186: `c.AbortWithStatusJSON(http.StatusForbidden, gin.H{...})` — frozen account.

**Write two helper functions inside the middleware package** (avoid inlining the envelope three times):

```go
// abortUnauthenticated writes the standard UNAUTHENTICATED envelope.
func abortUnauthenticated(c *gin.Context, reason, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"status":     string(cerrors.StatusUnauthenticated),
			"reason":     reason,
			"domain":     "api-manager",
			"message":    message,
			"request_id": RequestIDFromContext(c),
		},
	})
}

// abortForbidden writes the standard PERMISSION_DENIED envelope.
func abortForbidden(c *gin.Context, reason, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"status":     string(cerrors.StatusPermissionDenied),
			"reason":     reason,
			"domain":     "api-manager",
			"message":    message,
			"request_id": RequestIDFromContext(c),
		},
	})
}
```

Then replace call sites:
- Line 35: `abortUnauthenticated(c, "AUTHENTICATION_REQUIRED", "Authentication is required.")`
- Line 54: `abortUnauthenticated(c, "INVALID_CREDENTIALS", "The provided credentials are invalid.")`
- Line 186 (frozen account): `abortForbidden(c, "ACCOUNT_FROZEN", "This account is frozen. Contact support.")`

**Step 1: Update authenticate_test.go assertions**

Current test likely asserts `w.Code == 401` and doesn't inspect the body. Add body assertions using the new envelope shape — reuse the body-unmarshal pattern from Task 4's tests.

**Step 2: Run failing test**

Expected: new assertions fail because the old code doesn't write bodies.

**Step 3: Edit authenticate.go**

Add `cerrors` import, add the two helpers, replace the three call sites.

**Step 4: Run passing test**

```bash
cd bin-api-manager && go test ./lib/middleware/... -v
```

**Step 5: Commit**

```bash
git add bin-api-manager/lib/middleware/authenticate.go bin-api-manager/lib/middleware/authenticate_test.go
git commit -m "NOJIRA-api-error-response-0b

Migrate authenticate middleware to the new error envelope. Three call sites touched: missing auth header → UNAUTHENTICATED/AUTHENTICATION_REQUIRED, bad credentials → UNAUTHENTICATED/INVALID_CREDENTIALS, frozen account → PERMISSION_DENIED/ACCOUNT_FROZEN.

- bin-api-manager: Migrate lib/middleware/authenticate.go to the new error envelope"
```

---

## Task 8: Register the request-id middleware in `cmd/api-manager/main.go`

**Files:**
- Modify: `bin-api-manager/cmd/api-manager/main.go`

Insert `app.Use(middleware.RequestID())` BEFORE any other middleware (before CORS, before the custom lambda at line 215), so every request — including 404s and preflight — carries an ID.

**Step 1: Update tests**

There aren't many integration tests for main.go itself (it bootstraps the service). The middleware test from Task 2 covers the request-id behavior. This task is a pure wiring change — no new test.

**Step 2: Edit main.go**

```go
// After: app := gin.Default()
app.Use(middleware.RequestID())
// Then the existing: app.Use(cors.New(cors.Config{ ... }))
```

**Step 3: Smoke-test locally**

```bash
cd bin-api-manager && go build ./cmd/api-manager
```

Expected: build succeeds. (Full service boot requires DB/Rabbit/Redis — skip runtime test here, integration test will come from the api-validator in a companion PR.)

**Step 4: Commit**

```bash
git add bin-api-manager/cmd/api-manager/main.go
git commit -m "NOJIRA-api-error-response-0b

Register the request-id middleware at the top of the Gin chain so every request — including 404s and preflight — carries an X-Request-Id header and logs the correlation ID.

- bin-api-manager: Register request-id middleware in cmd/api-manager/main.go"
```

---

## Task 9: Add OpenAPI error schemas and named responses

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

Append the new `components.schemas` and `components.responses` blocks from design §9. Do NOT wire them into any endpoint yet — that's Task 10 (the spike).

**Step 1: Edit openapi.yaml**

Add:

```yaml
# Under components.schemas:
    ErrorBody:
      type: object
      required: [status, reason, domain, message, request_id]
      properties:
        status:
          type: string
          enum:
            - INVALID_ARGUMENT
            - UNAUTHENTICATED
            - PAYMENT_REQUIRED
            - PERMISSION_DENIED
            - NOT_FOUND
            - ALREADY_EXISTS
            - FAILED_PRECONDITION
            - RESOURCE_EXHAUSTED
            - UNAVAILABLE
            - INTERNAL
          description: Canonical error status. Maps 1:1 to HTTP status code.
        reason:
          type: string
          description: Specific VoIPbin reason code in UPPER_SNAKE (open-ended).
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
          example: req_ABCDEF0123456789
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
        error:
          $ref: '#/components/schemas/ErrorBody'

# Under components.responses (add the section if it does not yet exist):
  responses:
    BadRequest:
      description: Invalid request (INVALID_ARGUMENT).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    Unauthenticated:
      description: Authentication required (UNAUTHENTICATED).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    PaymentRequired:
      description: Payment required (PAYMENT_REQUIRED).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    PermissionDenied:
      description: Insufficient permission (PERMISSION_DENIED).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    NotFound:
      description: Resource not found (NOT_FOUND).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    Conflict:
      description: State conflict (ALREADY_EXISTS or FAILED_PRECONDITION).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    TooManyRequests:
      description: Rate or quota exceeded (RESOURCE_EXHAUSTED).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    Unavailable:
      description: Upstream unavailable (UNAVAILABLE).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    InternalError:
      description: Internal error (INTERNAL).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
```

**Step 2: Regenerate bin-openapi-manager types**

```bash
cd bin-openapi-manager && go generate ./...
```

**Step 3: Run bin-openapi-manager tests**

```bash
cd bin-openapi-manager && go test ./...
```

Expected: PASS.

**Step 4: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml bin-openapi-manager/$(any_generated_files)
git commit -m "NOJIRA-api-error-response-0b

Reserve the ErrorBody and ErrorResponse shapes plus nine named HTTP error responses. Schema landing only — no endpoint wiring yet.

- bin-openapi-manager: Add ErrorBody/ErrorResponse schemas and named HTTP error responses"
```

---

## Task 10: Spike — wire error responses into ONE endpoint, validate generator

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (one path only)
- Modify: `bin-api-manager/gens/openapi_server/gen.go` (regenerated)

Pick a truly trivial endpoint: `GET /v1.0/ping` is ideal if it exists. Otherwise `GET /v1.0/customer`.

**Goal:** verify `oapi-codegen` produces handler-compatible signatures when error responses are added to a path. If the generator emits per-path typed response variants that force handler signatures to change, we need to pivot (keep schemas for docs, don't wire per-path).

**Step 1: Modify the chosen path in openapi.yaml**

```yaml
/v1.0/ping:
  get:
    # ... existing fields unchanged ...
    responses:
      '200':
        # ... existing 200 response unchanged ...
      '401':
        $ref: '#/components/responses/Unauthenticated'
      '500':
        $ref: '#/components/responses/InternalError'
```

**Step 2: Regenerate both sides**

```bash
cd bin-openapi-manager && go generate ./...
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./...
```

**Step 3: Inspect `bin-api-manager/gens/openapi_server/gen.go`**

Check whether the handler signature for the ping endpoint changed. If it now expects a typed response struct return, we need to pivot. If it remains `func (s *server) GetPing(c *gin.Context)` (or equivalent), we're good.

**Decision point:**

- **If handler signature unchanged:** proceed — this path is viable for PR 1+ handler migrations.
- **If handler signature forces typed returns:** pivot in Task 11 — keep schemas for docs (ReDoc/Swagger will still render them), but don't wire `responses:` refs per-path. Document the pivot in the plan doc (this file) and in PR description.

**Step 4: Run full api-manager test**

```bash
cd bin-api-manager && go test ./server/... -v
```

Expected: PASS (spike is additive — should not break any existing test).

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml bin-api-manager/gens/openapi_server/gen.go bin-api-manager/go.mod bin-api-manager/go.sum
git commit -m "NOJIRA-api-error-response-0b

Wire error responses into GET /v1.0/ping as a code-generation compatibility spike. Confirms oapi-codegen output is unchanged (or documents pivot). No runtime behavior change.

- bin-openapi-manager: Add 401 and 500 error responses to GET /v1.0/ping
- bin-api-manager: Regenerate openapi_server/gen.go from updated spec"
```

---

## Task 11: RST docs — envelope spec + reason-code catalog

**Files:**
- Modify: `bin-api-manager/docsdev/source/restful_api.rst`
- Create: `bin-api-manager/docsdev/source/restful_api_errors.rst`
- Modify: `bin-api-manager/docsdev/source/index.rst` (add the new page to the ToC)
- Rebuild + commit: `bin-api-manager/docsdev/build/`

**Content of `restful_api.rst` additions:**

- Describe the error envelope (status, reason, domain, message, request_id) with a JSON example.
- Full Status → HTTP mapping table (the 10 canonical statuses).
- Cross-ref to `restful_api_errors.rst` for the reason-code catalog.
- AI Implementation Hint block per service-specific CLAUDE.md: "When a VoIPbin API returns a 4xx/5xx, inspect `error.reason` first, not `error.status`."

**Content of `restful_api_errors.rst` (skeleton — filled in as PRs 1+ ship):**

```rst
Error Reason Codes
==================

.. note:: **AI Context**

   Every 4xx/5xx response from the VoIPbin API contains an ``error.reason``
   field (UPPER_SNAKE). The catalog below groups reasons by ``domain``.
   Clients should branch on ``reason`` for debuggability; ``status`` maps
   1:1 to HTTP.

api-manager
-----------

.. list-table::
   :header-rows: 1
   :widths: 30 20 50

   * - Reason
     - HTTP
     - When
   * - ``AUTHENTICATION_REQUIRED``
     - 401
     - No token or access key was supplied; supply a valid JWT.
   * - ``INVALID_CREDENTIALS``
     - 401
     - Token/access key is invalid or expired; refresh and retry.
   * - ``ACCOUNT_FROZEN``
     - 403
     - Customer account is frozen; contact support.
   * - ``RATE_LIMIT_EXCEEDED``
     - 429
     - Client exceeded the rate limit for this endpoint.
   * - ``PERMISSION_DENIED``
     - 403
     - User does not have permission to access this resource.
   * - ``DIRECT_ACCESS_NOT_SUPPORTED``
     - 403
     - Endpoint requires a JWT auth flow that this direct access cannot provide.
   * - ``RESOURCE_NOT_FOUND``
     - 404
     - The requested resource does not exist or does not belong to the customer.
   * - ``INVALID_ARGUMENT``
     - 400
     - Request body or parameter is invalid.
   * - ``REQUEST_TIMEOUT``
     - 503
     - Upstream manager did not respond within the deadline.
   * - ``REQUEST_CANCELED``
     - 503
     - The request was canceled before completion.
   * - ``INTERNAL``
     - 500
     - An unexpected server error occurred; the ``request_id`` in the response
       body correlates with server logs.

call-manager
------------

.. list-table::
   :header-rows: 1
   :widths: 30 20 50

   * - Reason
     - HTTP
     - When

   (Catalog populated as PR 2 migrates call-group endpoints.)

billing-manager
---------------

(Catalog populated as PR 5 migrates billing endpoints.)
```

**Step 1: Edit restful_api.rst**

Add the envelope + status-mapping section. Cross-ref to the new catalog.

**Step 2: Create restful_api_errors.rst**

Paste the skeleton above.

**Step 3: Add to index.rst**

Find the ToC section and append `restful_api_errors`.

**Step 4: Rebuild Sphinx**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: build succeeds.

**Step 5: Force-add the build directory and commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-0b
git add bin-api-manager/docsdev/source/restful_api.rst bin-api-manager/docsdev/source/restful_api_errors.rst bin-api-manager/docsdev/source/index.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-api-error-response-0b

Document the new error envelope in restful_api.rst and introduce restful_api_errors.rst as the central reason-code catalog (populated incrementally in PRs 1+).

- bin-api-manager: Add error envelope spec and reason-code catalog page"
```

---

## Task 12: Full verification workflow — bin-openapi-manager

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-0b/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all 5 steps pass. If go.mod/go.sum mutated, commit.

```bash
# If needed:
git add bin-openapi-manager/go.mod bin-openapi-manager/go.sum
git commit -m "NOJIRA-api-error-response-0b

- bin-openapi-manager: Sync go.mod/go.sum after generator regeneration"
```

---

## Task 13: Full verification workflow — bin-api-manager

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-0b/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Also run with race detector (per PR 0a lesson):

```bash
cd bin-api-manager && go test -race ./...
```

Expected: all 6 steps pass. Commit any go.mod/go.sum/gen.go drift.

---

## Task 14: Pre-push main conflict check, push, open PR

**Step 1: Fetch main and check conflicts**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-0b
git fetch origin main
git log --oneline HEAD..origin/main
```

**If PR 0a has already merged:** `git log` will show new commits on main. Rebase:

```bash
git rebase origin/main
```

Resolve any conflicts, re-run Task 13 verification.

**If PR 0a has NOT yet merged:** branch is still stacked on PR 0a. That's fine; GitHub will rebase when PR 0a squash-merges.

**Step 2: Push**

```bash
git push -u origin NOJIRA-api-error-response-0b
```

**Step 3: Open PR**

```bash
gh pr create --title "NOJIRA-api-error-response-0b" --body "$(cat <<'EOF'
Wires the shared VoipbinError vocabulary (from PR 0a) into the api-manager HTTP layer and migrates the two middleware sites that already emit HTTP errors. Zero handler-file migration — that begins in PR 1.

Depends on #799 (PR 0a). This branch is stacked on NOJIRA-api-error-response-codes; rebase onto main after PR 0a merges.

See design doc: docs/plans/2026-04-24-api-error-response-codes-design.md
See PR 0b plan: docs/plans/2026-04-24-api-error-response-0b-plan.md

- bin-common-handler: Promote httpStatusFor to exported HTTPStatusFor (single source of truth for Status-to-HTTP)
- bin-api-manager: Add pkg/serviceerrors/sentinels.go (6 sentinel errors for the translator)
- bin-api-manager: Add lib/middleware/request_id.go (30-char base32 request IDs, echoes inbound X-Request-Id)
- bin-api-manager: Add server/error.go (abortWithError, abortWithServiceError, assertErrorResponse test helper)
- bin-api-manager: Add server/error_translate.go (typed-passthrough → sentinel → transport → substring → INTERNAL fallback chain with panic safety)
- bin-api-manager: Migrate lib/middleware/ratelimit.go to new envelope (RATE_LIMIT_EXCEEDED)
- bin-api-manager: Migrate lib/middleware/authenticate.go to new envelope (AUTHENTICATION_REQUIRED, INVALID_CREDENTIALS, ACCOUNT_FROZEN)
- bin-api-manager: Register request-id middleware in cmd/api-manager/main.go
- bin-openapi-manager: Add ErrorBody, ErrorResponse, and 9 named error responses (with reserved details array for forward compatibility)
- bin-openapi-manager: Wire error responses into GET /v1.0/ping as a codegen compatibility spike
- bin-api-manager: Add docsdev/source/restful_api_errors.rst reason-code catalog (populated as PRs 1+ ship) and update restful_api.rst with the envelope spec

Reviewer note: PR 0b does NOT migrate any handler file. 1047 c.AbortWithStatus sites remain unchanged. Handler migration begins in PR 1 (auth & identity group).
EOF
)"
```

---

## PR 0b success criteria

- [ ] All 14 tasks committed.
- [ ] `go test ./... -race` passes inside both `bin-common-handler`, `bin-openapi-manager`, and `bin-api-manager`.
- [ ] `golangci-lint run -v --timeout 5m` is clean in all three.
- [ ] Spike confirmed codegen compatibility OR pivot documented.
- [ ] Sphinx HTML rebuilt and committed via `git add -f`.
- [ ] 0 handler-file migrations (server/*.go unchanged).
- [ ] 1047 `c.AbortWithStatus` sites remain untouched in `server/` (deliberate — PR 1+).
- [ ] PR description clearly states this depends on PR 0a.

---

## Known risks

1. **`oapi-codegen` spike fails.** Handled in Task 10 decision point: pivot to docs-only components.
2. **Middleware import cycle.** `lib/middleware` cannot depend on `server`, so error envelopes are inlined in two places (rate-limit + authenticate). Acceptable for PR 0b; follow-up PR can refactor both sites to a shared `pkg/httperr` package if drift becomes a problem.
3. **Sentinel message text drift.** Sentinels in Task 1 use specific strings (e.g. `"permission denied"`). Each servicehandler callsite in PRs 1+ must return the sentinel, NOT a `fmt.Errorf` with its own string — the substring fallback in Task 5 catches the old style, but once migrated, the sentinel path is exact and fast.
4. **ULID dependency avoided.** Plan originally said `github.com/oklog/ulid/v2`; Task 2 switched to `crypto/rand + base32` to avoid a new external dep. ID format is still opaque 26-char base32 with `req_` prefix, satisfying all downstream needs.
5. **Request-ID not yet propagated to RPC.** Task 2 puts the ID into `c.Request.Context()` and a logrus field, but the bin-common-handler `requesthandler` does not yet read it into `sock.Request.RequestID`. That wiring is a follow-up for PR 0b-part-2 or PR 1 — flag in the PR body.

---

## Execution handoff

Plan saved. Two execution options:

**1. Subagent-driven (this session)** — Dispatch a fresh subagent per task, review between tasks, iterate until all 14 tasks complete and PR is open. Uses `superpowers:subagent-driven-development`.

**2. Parallel session (separate)** — Open a new session in this worktree; it uses `superpowers:executing-plans` for batched execution.

Blocker to actually starting execution: PR 0a is still in review. If review requires structural changes to the shared types, some of PR 0b's tasks may need revision. Options:

- **Execute now (stacked).** Start executing tasks against the current stacked branch; rebase + patch if PR 0a review surfaces issues.
- **Wait for PR 0a merge.** Hold execution until PR 0a lands; then rebase this branch onto main and execute.

Which approach?
