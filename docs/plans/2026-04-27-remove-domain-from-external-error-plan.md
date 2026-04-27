# Remove `domain` from External HTTP Error Responses — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Stop emitting the internal microservice name (`Domain` field on `VoipbinError`) in external HTTP error JSON bodies returned by `bin-api-manager`, while preserving it for internal RPC and server-side logs.

**Architecture:** Introduce a single boundary helper `bin-api-manager/lib/apierror/EnvelopeFor` that produces the external JSON shape *without* `domain`. Refactor 4 open-coded envelope sites (`server/error.go`, two in `lib/middleware/authenticate.go`, `lib/middleware/ratelimit.go`) to route through it. Update OpenAPI schema, RST docs, and a CI grep guard to prevent re-leaks.

**Tech Stack:** Go (gin-gonic/gin), gomock for unit tests, OpenAPI 3.0 (oapi-codegen), Sphinx (RST docs), bash for the CI guard.

**Reference:** Design doc at `docs/plans/2026-04-27-remove-domain-from-external-error-design.md` (committed in this branch). Read §4.3 for the four sites, §5.1 for RST regrouping, §6.1–§6.5 for the test plan.

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error` (branch `NOJIRA-remove-domain-from-external-error`). All work happens here. NEVER edit files in `~/gitvoipbin/monorepo` directly.

**Branch & commit format reminder:** Each commit's title MUST be `NOJIRA-remove-domain-from-external-error` (matches the branch). Body lists affected projects with `bin-<service>:` prefixes. No AI attribution. See root `CLAUDE.md` for full rules.

---

## Phase 1 — Create the `apierror` helper package (TDD)

### Task 1.1: Write the failing unit test for `EnvelopeFor` — full struct case

**Files:**
- Create: `bin-api-manager/lib/apierror/envelope_test.go`

**Step 1: Write the failing test**

```go
package apierror

import (
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

func TestEnvelopeFor_FullVoipbinError_OmitsDomain(t *testing.T) {
	e := cerrors.NotFound(commonoutline.ServiceNameCallManager, "CALL_NOT_FOUND", "The call was not found.")

	body := EnvelopeFor(e, "req-abc-123")

	outer, ok := body["error"].(gin.H)
	if !ok {
		t.Fatalf("body.error is not gin.H: %+v", body)
	}
	if outer["status"] != "NOT_FOUND" {
		t.Errorf("status = %v, want NOT_FOUND", outer["status"])
	}
	if outer["reason"] != "CALL_NOT_FOUND" {
		t.Errorf("reason = %v, want CALL_NOT_FOUND", outer["reason"])
	}
	if outer["message"] != "The call was not found." {
		t.Errorf("message = %v", outer["message"])
	}
	if outer["request_id"] != "req-abc-123" {
		t.Errorf("request_id = %v", outer["request_id"])
	}
	if _, present := outer["domain"]; present {
		t.Errorf("domain key MUST be absent, got: %v", outer["domain"])
	}
	if _, present := outer["details"]; present {
		t.Errorf("details key MUST be absent for empty Details, got: %v", outer["details"])
	}
}
```

**Step 2: Run the test to confirm it fails to compile**

Run:
```bash
cd bin-api-manager && go test ./lib/apierror/...
```

Expected: compilation failure (`undefined: EnvelopeFor`, package does not exist).

---

### Task 1.2: Create the `apierror` package with the helper

**Files:**
- Create: `bin-api-manager/lib/apierror/envelope.go`

**Step 1: Write the helper**

```go
// Package apierror builds the external HTTP error envelope for the
// VoIPbin public API. The envelope intentionally omits the internal
// Domain (originating service name) carried by VoipbinError — that field
// is internal-only and must not cross the public API boundary. The
// VoipbinError.Domain field stays available for server-side logs and
// internal RPC; only this external serialization strips it.
package apierror

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
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

**Step 2: Run the test to confirm it passes**

Run:
```bash
cd bin-api-manager && go test ./lib/apierror/... -run TestEnvelopeFor_FullVoipbinError_OmitsDomain -v
```

Expected: `--- PASS`.

---

### Task 1.3: Add the remaining unit-test cases

**Files:**
- Modify: `bin-api-manager/lib/apierror/envelope_test.go` (append)

**Step 1: Append the additional cases**

```go
func TestEnvelopeFor_DetailsIncluded(t *testing.T) {
	e := cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_FIELD", "Validation failed.")
	e.Details = []map[string]any{
		{"field": "destination", "issue": "must be E.164"},
	}

	body := EnvelopeFor(e, "req-1")

	outer := body["error"].(gin.H)
	details, ok := outer["details"].([]map[string]any)
	if !ok {
		t.Fatalf("details key missing or wrong type: %+v", outer)
	}
	if len(details) != 1 || details[0]["field"] != "destination" {
		t.Errorf("details payload not preserved: %+v", details)
	}
	if _, present := outer["domain"]; present {
		t.Errorf("domain key MUST be absent")
	}
}

func TestEnvelopeFor_NilDetails_OmitsKey(t *testing.T) {
	e := cerrors.NotFound(commonoutline.ServiceNameCallManager, "CALL_NOT_FOUND", "x")
	e.Details = nil

	body := EnvelopeFor(e, "req-1")

	outer := body["error"].(gin.H)
	if _, present := outer["details"]; present {
		t.Errorf("details key MUST be omitted when nil")
	}
}

func TestEnvelopeFor_NilVoipbinError_FallsBackToInternal(t *testing.T) {
	body := EnvelopeFor(nil, "req-fallback")

	outer := body["error"].(gin.H)
	if outer["status"] != "INTERNAL" {
		t.Errorf("fallback status = %v, want INTERNAL", outer["status"])
	}
	if outer["reason"] != "INTERNAL" {
		t.Errorf("fallback reason = %v, want INTERNAL", outer["reason"])
	}
	if outer["request_id"] != "req-fallback" {
		t.Errorf("request_id not preserved: %v", outer["request_id"])
	}
	if _, present := outer["domain"]; present {
		t.Errorf("domain key MUST be absent on fallback")
	}
}

func TestEnvelopeFor_EmptyRequestID(t *testing.T) {
	e := cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "x")

	body := EnvelopeFor(e, "")

	outer := body["error"].(gin.H)
	if outer["request_id"] != "" {
		t.Errorf("expected empty request_id, got %v", outer["request_id"])
	}
}
```

**Step 2: Run all unit tests**

Run:
```bash
cd bin-api-manager && go test ./lib/apierror/... -v
```

Expected: 4 tests PASS (the original + 3 new ones).

---

### Task 1.4: Commit Phase 1

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
git add bin-api-manager/lib/apierror/
git commit -m "$(cat <<'EOF'
NOJIRA-remove-domain-from-external-error

- bin-api-manager: Add lib/apierror package with EnvelopeFor helper that builds external HTTP error envelope without internal Domain field
- bin-api-manager: Add unit tests covering full VoipbinError, Details inclusion/omission, nil-fallback to INTERNAL, and empty request_id; every case asserts absence of "domain" key
EOF
)"
```

---

## Phase 2 — Refactor `server/error.go` (the main path)

### Task 2.1: Refactor `abortWithError` to use the helper

**Files:**
- Modify: `bin-api-manager/server/error.go`

**Step 1: Rewrite the file**

Replace the body of `abortWithError`. The new file should be:

```go
package server

import (
	"monorepo/bin-api-manager/lib/apierror"
	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gin-gonic/gin"
)

// abortWithError writes the VoipbinError as a JSON body, sets the
// correct HTTP status code, and aborts the Gin context. The request
// ID is read from middleware.RequestIDFromContext, so that middleware
// must run before any handler that calls this.
//
// The external envelope omits the internal Domain field — see
// bin-api-manager/lib/apierror for the boundary.
//
// A nil VoipbinError falls back to a StatusInternal response so the
// helper never panics on a caller oversight (handled inside
// apierror.EnvelopeFor).
func abortWithError(c *gin.Context, e *cerrors.VoipbinError) {
	status := cerrors.StatusInternal
	if e != nil {
		status = e.Status
	}
	c.AbortWithStatusJSON(
		cerrors.HTTPStatusFor(status),
		apierror.EnvelopeFor(e, middleware.RequestIDFromContext(c)),
	)
}

// abortWithServiceError runs any error returned from servicehandler
// through the translator (see server/error_translate.go), then aborts
// with the resulting VoipbinError.
func abortWithServiceError(c *gin.Context, err error) {
	abortWithError(c, translateToVoipbinError(err))
}
```

**Step 2: Run server tests to check the change compiles and the basic tests still pass**

Run:
```bash
cd bin-api-manager && go test ./server/... -run TestAbortWithErrorSetsStatusAndBody -v
```

Expected: **FAIL** (the test currently asserts `body.Error.Domain == "call-manager"`; that field is no longer emitted). This is expected — Task 2.2 fixes it.

---

### Task 2.2: Update `server/error_test.go` to drop `Domain` from the envelope test struct and assertions; update `assertErrorResponse` signature

**Files:**
- Modify: `bin-api-manager/server/error_test.go`

**Step 1: Drop `Domain` field from local struct + `body.Error.Domain` assertions in `TestAbortWithErrorSetsStatusAndBody`**

In `TestAbortWithErrorSetsStatusAndBody` (lines 18–60), change the body struct and remove the domain check:

```go
	var body struct {
		Error struct {
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
```

Delete lines 51–53 (the `body.Error.Domain != "call-manager"` block). Add an explicit absence-of-domain assertion immediately after the unmarshal:

```go
	// domain MUST NOT be present in the external envelope.
	if strings.Contains(w.Body.String(), `"domain"`) {
		t.Errorf("domain key MUST be absent from external response; body=%s", w.Body.String())
	}
```

**Step 2: Update `assertErrorResponse` helper signature — drop `wantDomain` parameter**

Replace the helper at lines 175–208 with:

```go
// assertErrorResponse is a test helper shared across handler tests. It
// asserts the HTTP status code matches the canonical Status AND the
// response body's status and reason fields match the expected values,
// plus verifies request_id is present so handler tests catch missing
// RequestID middleware registration. The external envelope intentionally
// does NOT include a "domain" field — see lib/apierror.
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, wantStatus cerrors.Status, wantReason string) {
	t.Helper()
	if got, want := w.Code, cerrors.HTTPStatusFor(wantStatus); got != want {
		t.Errorf("status code = %d want %d", got, want)
	}
	var body struct {
		Error struct {
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			RequestID string `json:"request_id"`
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
	if body.Error.RequestID == "" {
		t.Error("request_id missing — RequestID middleware must run before the handler")
	}
	if strings.Contains(w.Body.String(), `"domain"`) {
		t.Errorf("domain key MUST be absent from external response; body=%s", w.Body.String())
	}
}
```

**Step 3: Update `TestAssertErrorResponseHelper` (lines 110–122) to match new signature**

```go
func TestAssertErrorResponseHelper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "bad id"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}
```

**Step 4: Verify the import for `commonoutline` is still needed**

After dropping the `wantDomain` parameter, `commonoutline` may no longer be used in this file. Check by re-reading the file and removing the import if unused. Note: it's still used by other test bodies (e.g., `cerrors.NotFound(commonoutline.ServiceNameCallManager, ...)`) so likely stays.

**Step 5: Run only the error_test.go tests**

Run:
```bash
cd bin-api-manager && go test ./server/... -run 'TestAbortWithError|TestAssertErrorResponse|TestNoRoute' -v
```

Expected: most pass; some `assertErrorResponse` callers in OTHER test files will now fail to compile (the signature changed). Task 2.3 fixes this.

---

### Task 2.3: Mass-update all 97 callers of `assertErrorResponse` across `bin-api-manager/server/`

**Files:**
- Modify: 43 test files in `bin-api-manager/server/` (all that call `assertErrorResponse`)

**Step 1: Identify every caller**

Run:
```bash
cd bin-api-manager && grep -rln "assertErrorResponse" server/ --include="*_test.go"
```

Expected: 43 file paths.

**Step 2: Mechanical sed transform — drop the trailing `, commonoutline.ServiceName...` argument**

The pattern is always:
```
assertErrorResponse(t, w, <status>, "<REASON>", commonoutline.ServiceName<X>)
```

Change to:
```
assertErrorResponse(t, w, <status>, "<REASON>")
```

Run from the worktree root:
```bash
cd bin-api-manager && grep -rl "assertErrorResponse" server/ --include="*_test.go" | xargs sed -i -E 's/(assertErrorResponse\([^)]*"[A-Z_]+"),\s*commonoutline\.ServiceName[A-Za-z]+\)/\1)/g'
```

**Step 3: Verify no callers were missed**

Run:
```bash
cd bin-api-manager && grep -rn "assertErrorResponse.*commonoutline" server/ --include="*_test.go"
```

Expected: empty (no matches). If any remain, hand-fix them.

**Step 4: Verify the call-site count is unchanged (97)**

Run:
```bash
cd bin-api-manager && grep -rn "assertErrorResponse" server/ --include="*_test.go" | wc -l
```

Expected: `97`.

**Step 5: Remove now-unused `commonoutline` imports if any test file no longer needs them**

Run:
```bash
cd bin-api-manager && go build ./server/... 2>&1 | grep -E "imported and not used|undefined" | head
```

If any unused-import errors appear, remove them. The Go compiler will pinpoint each file/line.

**Step 6: Run the full server test package**

Run:
```bash
cd bin-api-manager && go test ./server/... -v 2>&1 | tail -30
```

Expected: PASS, no failures, no compile errors. Watch for any test that has its own local `body.Error.Domain` assertion (not via the helper) — fix those by hand.

**Step 7: Sweep for any other stragglers asserting on `domain` in server tests**

Run:
```bash
cd bin-api-manager && grep -rn '"domain"' server/ --include="*_test.go"
```

Expected: no remaining assertion uses (or only the absence-of-domain assertions added in Task 2.2). If any positive `wrong domain:` checks remain, drop them.

---

### Task 2.4: Update `error_translate_test.go` if it asserts on `domain`

**Files:**
- Possibly modify: `bin-api-manager/server/error_translate_test.go`

**Step 1: Check whether it asserts on domain in the response body**

Run:
```bash
grep -n "domain" bin-api-manager/server/error_translate_test.go
```

If matches reference the **typed VoipbinError struct's `Domain` field** (e.g., `e.Domain == "..."`), leave them — internal flow unchanged. If matches reference the **HTTP body** (e.g., `body.Error.Domain`), drop them per the same pattern as `error_test.go`.

**Step 2: Run the translate tests**

Run:
```bash
cd bin-api-manager && go test ./server/... -run TestTranslate -v
```

Expected: PASS.

---

### Task 2.5: Run full bin-api-manager test suite to confirm Phase 2 is green

Run:
```bash
cd bin-api-manager && go test ./... 2>&1 | tail -10
```

Expected: PASS overall. If failures reference middleware tests, that's expected — they are addressed in Phase 3.

---

### Task 2.6: Commit Phase 2

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
git add bin-api-manager/server/
git commit -m "$(cat <<'EOF'
NOJIRA-remove-domain-from-external-error

- bin-api-manager: Refactor server/error.go abortWithError to route through lib/apierror.EnvelopeFor; external envelope no longer includes the internal "domain" field
- bin-api-manager: Drop wantDomain parameter from assertErrorResponse helper and update all 97 call sites across 43 test files
- bin-api-manager: Add absence-of-domain assertions in error_test.go and assertErrorResponse to catch any future re-leak
EOF
)"
```

---

## Phase 3 — Refactor middleware sites (`authenticate.go` × 2, `ratelimit.go`)

### Task 3.1: Refactor `abortUnauthenticated`

**Files:**
- Modify: `bin-api-manager/lib/middleware/authenticate.go` (function around line 280)

**Step 1: Replace the function body**

Find `func abortUnauthenticated` (currently around line 280). Keep the public signature `abortUnauthenticated(c *gin.Context, reason, message string)` so the two callers (around lines 37 and 56) need no diff. Replace the body:

```go
// abortUnauthenticated writes the standard UNAUTHENTICATED envelope.
// The external envelope omits the internal Domain field — see
// bin-api-manager/lib/apierror.
func abortUnauthenticated(c *gin.Context, reason, message string) {
	e := cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, reason, message)
	c.AbortWithStatusJSON(
		cerrors.HTTPStatusFor(e.Status),
		apierror.EnvelopeFor(e, RequestIDFromContext(c)),
	)
}
```

**Step 2: Add the `apierror` import**

Add to the imports block at the top of `authenticate.go`:

```go
"monorepo/bin-api-manager/lib/apierror"
```

`cerrors` and `commonoutline` are already imported.

**Step 3: Compile-check**

Run:
```bash
cd bin-api-manager && go build ./lib/middleware/...
```

Expected: success.

---

### Task 3.2: Refactor `isFrozenAccountBlocked`

**Files:**
- Modify: `bin-api-manager/lib/middleware/authenticate.go` (around lines 195–215)

**Step 1: Replace the inline `c.AbortWithStatusJSON` block**

Find the existing block (lines 202–211 in the current file):

```go
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"status":     string(cerrors.StatusPermissionDenied),
			"reason":     "ACCOUNT_FROZEN",
			"domain":     string(commonoutline.ServiceNameAPIManager),
			"message":    "This account is frozen. Contact support.",
			"request_id": RequestIDFromContext(c),
			"details":    details,
		},
	})
```

Replace with:

```go
	e := cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "ACCOUNT_FROZEN", "This account is frozen. Contact support.")
	e.Details = details
	c.AbortWithStatusJSON(
		cerrors.HTTPStatusFor(e.Status),
		apierror.EnvelopeFor(e, RequestIDFromContext(c)),
	)
```

**Step 2: Verify `details` variable type matches `[]map[string]any`**

The existing `details` value (currently `[]map[string]any{{...}}`) must be assignment-compatible with `VoipbinError.Details []map[string]any`. Reading the file confirms it is — it's already constructed as `[]map[string]any{...}`. No change needed.

**Step 3: Verify `http` import may now be unused in this file**

After replacing the inline block, check if `net/http` is still imported but unused. If so, remove it. The cleanest check:

```bash
cd bin-api-manager && go build ./lib/middleware/... 2>&1
```

Address any "imported and not used" error.

---

### Task 3.3: Refactor `lib/middleware/ratelimit.go` rate-limit branch

**Files:**
- Modify: `bin-api-manager/lib/middleware/ratelimit.go` (around lines 78–92)

**Step 1: Replace the inline envelope**

Find the rate-limit-exceeded block:

```go
		if !limiter.Allow() {
			// Inline envelope construction — lib/middleware cannot import
			// the server package (would create an import cycle), so we
			// build the same shape as server.abortWithError here.
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"status":     string(cerrors.StatusResourceExhausted),
					"reason":     "RATE_LIMIT_EXCEEDED",
					"domain":     string(commonoutline.ServiceNameAPIManager),
					"message":    "Too many requests. Please try again later.",
					"request_id": RequestIDFromContext(c),
				},
			})
			return
		}
```

Replace with:

```go
		if !limiter.Allow() {
			// Build the canonical external envelope. The internal Domain
			// field is omitted by lib/apierror — see envelope.go.
			e := cerrors.ResourceExhausted(commonoutline.ServiceNameAPIManager, "RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later.")
			c.AbortWithStatusJSON(
				cerrors.HTTPStatusFor(e.Status),
				apierror.EnvelopeFor(e, RequestIDFromContext(c)),
			)
			return
		}
```

**Step 2: Add the `apierror` import; remove `net/http` if unused**

Add `"monorepo/bin-api-manager/lib/apierror"` to imports. Check whether `net/http` is still used elsewhere in the file; if not, remove.

**Step 3: Compile-check**

```bash
cd bin-api-manager && go build ./lib/middleware/...
```

Expected: success.

---

### Task 3.4: Update `authenticate_test.go` — drop `Domain` from `assertAuthErrorEnvelope` and any frozen-account test

**Files:**
- Modify: `bin-api-manager/lib/middleware/authenticate_test.go`

**Step 1: Update `assertAuthErrorEnvelope` (around lines 360–390)**

Replace the helper:

```go
// assertAuthErrorEnvelope decodes the response body and asserts the
// standard error envelope fields used by the Authenticate middleware.
// The external envelope intentionally does NOT include a "domain" field —
// see bin-api-manager/lib/apierror.
func assertAuthErrorEnvelope(t *testing.T, body []byte, wantStatus, wantReason string) {
	t.Helper()
	var decoded struct {
		Error struct {
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v; body: %s", err, string(body))
	}
	if decoded.Error.Status != wantStatus {
		t.Errorf("wrong status: got %q, want %q", decoded.Error.Status, wantStatus)
	}
	if decoded.Error.Reason != wantReason {
		t.Errorf("wrong reason: got %q, want %q", decoded.Error.Reason, wantReason)
	}
	if decoded.Error.Message == "" {
		t.Error("message missing")
	}
	if decoded.Error.RequestID == "" {
		t.Error("request_id missing")
	}
	if strings.Contains(string(body), `"domain"`) {
		t.Errorf("domain key MUST be absent; body=%s", string(body))
	}
}
```

Add `"strings"` to the imports if not present.

**Step 2: Update `Test_isFrozenAccountBlocked` (around line 784)**

If this test asserts on `body.Error.Domain` directly, drop the assertion. Add an explicit `if strings.Contains(...)` absence check. Verify the test still asserts on `details` content (must be preserved).

**Step 3: Run middleware tests**

```bash
cd bin-api-manager && go test ./lib/middleware/... -v 2>&1 | tail -20
```

Expected: PASS.

---

### Task 3.5: Update `ratelimit_test.go` — drop `Domain` assertion in `TestRateLimit_EnvelopeShape`

**Files:**
- Modify: `bin-api-manager/lib/middleware/ratelimit_test.go` (around lines 100–128)

**Step 1: Drop `Domain` field from local struct + assertion**

Change the struct (around line 100):

```go
	var body struct {
		Error struct {
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
```

Delete the `if body.Error.Domain != "api-manager"` block (around lines 119–121).

**Step 2: Add absence-of-domain assertion**

```go
	if strings.Contains(w2.Body.String(), `"domain"`) {
		t.Errorf("domain key MUST be absent; body=%s", w2.Body.String())
	}
```

Add `"strings"` import if not present.

**Step 3: Run rate-limit tests**

```bash
cd bin-api-manager && go test ./lib/middleware/... -run TestRateLimit -v
```

Expected: PASS.

---

### Task 3.6: Run full middleware tests

```bash
cd bin-api-manager && go test ./lib/middleware/... -v 2>&1 | tail -10
```

Expected: PASS, no failures.

---

### Task 3.7: Commit Phase 3

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
git add bin-api-manager/lib/middleware/
git commit -m "$(cat <<'EOF'
NOJIRA-remove-domain-from-external-error

- bin-api-manager: Refactor lib/middleware/authenticate.go abortUnauthenticated and isFrozenAccountBlocked to route through lib/apierror.EnvelopeFor; external envelope no longer includes "domain"
- bin-api-manager: Refactor lib/middleware/ratelimit.go rate-limit branch to route through lib/apierror.EnvelopeFor
- bin-api-manager: Update assertAuthErrorEnvelope and TestRateLimit_EnvelopeShape to drop domain assertions and add absence-of-domain regression checks
EOF
)"
```

---

## Phase 4 — Integration tests (full handler chain)

### Task 4.1: Add the integration test file

**Files:**
- Create: `bin-api-manager/lib/apierror/integration_test.go`

**Step 1: Write the integration tests**

```go
package apierror_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monorepo/bin-api-manager/lib/apierror"
	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

// assertNoDomainInBody is the load-bearing assertion for the entire
// design: every external HTTP error body must omit the "domain" key.
func assertNoDomainInBody(t *testing.T, body string) {
	t.Helper()
	if strings.Contains(body, `"domain"`) {
		t.Errorf("domain key MUST be absent from external response; body=%s", body)
	}
}

func TestIntegration_TypedErrorPath_OmitsDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/x", func(c *gin.Context) {
		e := cerrors.NotFound(commonoutline.ServiceNameCallManager, "CALL_NOT_FOUND", "x")
		c.AbortWithStatusJSON(cerrors.HTTPStatusFor(e.Status), apierror.EnvelopeFor(e, middleware.RequestIDFromContext(c)))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d want 404", w.Code)
	}
	assertNoDomainInBody(t, w.Body.String())
}

func TestIntegration_NilFallback_OmitsDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/x", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.EnvelopeFor(nil, middleware.RequestIDFromContext(c)))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d want 500", w.Code)
	}
	var body struct {
		Error struct {
			Status string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Status != "INTERNAL" {
		t.Errorf("fallback status = %q want INTERNAL", body.Error.Status)
	}
	assertNoDomainInBody(t, w.Body.String())
}

func TestIntegration_DetailsPreserved_OmitsDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/x", func(c *gin.Context) {
		e := cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "ACCOUNT_FROZEN", "frozen")
		e.Details = []map[string]any{{"recovery_endpoint": "DELETE /auth/unregister"}}
		c.AbortWithStatusJSON(cerrors.HTTPStatusFor(e.Status), apierror.EnvelopeFor(e, middleware.RequestIDFromContext(c)))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"recovery_endpoint"`) {
		t.Errorf("details payload not preserved; body=%s", w.Body.String())
	}
	assertNoDomainInBody(t, w.Body.String())
}

// TestIntegration_PanicRecovery_CurrentBehavior is a guardrail. Today
// gin.Recovery() returns an empty body on panic — no envelope, no
// domain (no leak). If this test ever fails because a future Recovery
// middleware emits an envelope, the assertion must be updated to verify
// no "domain" key in that envelope (the design explicitly defers the
// envelope-on-panic work as a follow-up; see design doc §3.2).
func TestIntegration_PanicRecovery_CurrentBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default() // gin.Default installs gin.Recovery()
	r.Use(middleware.RequestID())
	r.GET("/x", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d want 500", w.Code)
	}
	// Current behavior: empty body. If this changes to an envelope,
	// update the assertion to also check no "domain" key.
	if w.Body.Len() != 0 {
		t.Logf("WARNING: gin.Recovery now emits a body (%q). Verify no domain leak and update this assertion.", w.Body.String())
		assertNoDomainInBody(t, w.Body.String())
	}
}
```

**Step 2: Run integration tests**

```bash
cd bin-api-manager && go test ./lib/apierror/... -run TestIntegration -v
```

Expected: 4 PASS.

---

### Task 4.2: Commit Phase 4

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
git add bin-api-manager/lib/apierror/integration_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-remove-domain-from-external-error

- bin-api-manager: Add lib/apierror integration tests that drive full gin handler chains end-to-end and assert no "domain" key in response bodies for typed-error, nil-fallback, details-preserved, and panic-recovery (current-behavior) paths
EOF
)"
```

---

## Phase 5 — CI grep guard

### Task 5.1: Create the grep-guard script

**Files:**
- Create: `scripts/check-error-envelope.sh`

**Step 1: Write the script**

```bash
#!/usr/bin/env bash
# scripts/check-error-envelope.sh — guards against the internal
# Domain field re-leaking into external HTTP error responses.
#
# The bin-api-manager error envelope is built exclusively by
# bin-api-manager/lib/apierror/EnvelopeFor. Any "domain": literal
# inside an open-coded gin.H{} envelope under server/, lib/middleware/,
# or lib/service/ is a regression. Test files are exempt because they
# may legitimately mention "domain" in absence-of-domain assertions.
#
# Invocation: from the monorepo root.
#   make lint-error-envelope
#
# Exits non-zero on any match.

set -euo pipefail

SCOPES=(
  bin-api-manager/server
  bin-api-manager/lib/middleware
  bin-api-manager/lib/service
)

MATCHES=0

for scope in "${SCOPES[@]}"; do
  if [[ ! -d "$scope" ]]; then
    continue
  fi
  while IFS= read -r line; do
    MATCHES=$((MATCHES + 1))
    echo "$line" >&2
  done < <(grep -rEn '"domain"\s*:' "$scope" --include="*.go" --exclude="*_test.go" || true)
done

if [[ $MATCHES -gt 0 ]]; then
  echo "" >&2
  echo "FAIL: found $MATCHES open-coded \"domain\": literal(s) in api-manager non-test files." >&2
  echo "The external HTTP error envelope MUST NOT include \"domain\". Use bin-api-manager/lib/apierror.EnvelopeFor." >&2
  exit 1
fi

echo "OK: no open-coded \"domain\": literals in api-manager error sites."
```

**Step 2: Make it executable**

```bash
chmod +x scripts/check-error-envelope.sh
```

**Step 3: Run it from the worktree root**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
./scripts/check-error-envelope.sh
```

Expected: `OK: no open-coded "domain": literals in api-manager error sites.`

---

### Task 5.2: Wire it into the top-level Makefile

**Files:**
- Modify: `Makefile`

**Step 1: Add the new target and update `.PHONY`**

After the existing `lint-docs` target, append:

```makefile
.PHONY: lint-error-envelope

lint-error-envelope:
	@./scripts/check-error-envelope.sh
```

Also update `.PHONY` line near the top to include the new target:

```makefile
.PHONY: lint-docs lint-error-envelope
```

**Step 2: Run the new target**

```bash
make lint-error-envelope
```

Expected: `OK: no open-coded "domain": literals in api-manager error sites.`

**Step 3: Verify the guard catches a real regression (manual smoke, do not commit)**

Temporarily inject a regression to confirm the guard works:

```bash
sed -i 's|"status":|"domain": "api-manager",\n\t\t"status":|' bin-api-manager/server/error.go
make lint-error-envelope || echo "guard correctly fails"
git checkout -- bin-api-manager/server/error.go
make lint-error-envelope
```

Expected: middle command exits non-zero with the FAIL message; final command passes again. Do NOT commit the temporary regression.

---

### Task 5.3: Commit Phase 5

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
git add scripts/check-error-envelope.sh Makefile
git commit -m "$(cat <<'EOF'
NOJIRA-remove-domain-from-external-error

- monorepo: Add scripts/check-error-envelope.sh CI guard that exits non-zero on any open-coded "domain": literal in bin-api-manager/server/, lib/middleware/, or lib/service/ non-test files
- monorepo: Wire lint-error-envelope target into the top-level Makefile
EOF
)"
```

---

## Phase 6 — OpenAPI schema

### Task 6.1: Drop `domain` from `ErrorBody` schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Edit the schema**

Find the `ErrorBody` schema (search for `ErrorBody:` — should be around line 6937).

Change the `required:` list from:
```yaml
      required: [status, reason, domain, message, request_id]
```
to:
```yaml
      required: [status, reason, message, request_id]
```

Delete the `domain` property block (currently around lines 6959–6962):
```yaml
        domain:
          type: string
          description: Originating manager service.
          example: call-manager
```

**Step 2: Regenerate types in bin-openapi-manager**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./...
```

**Step 3: Run bin-openapi-manager tests + lint**

```bash
cd bin-openapi-manager && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS.

---

### Task 6.2: Update bin-api-manager generated types and tests

**Files:**
- Modify: `bin-api-manager/gens/openapi_server/gen.go` (regenerated, not hand-edited)
- Possibly modify: any handler that referenced the now-removed `Domain` field of the generated `ErrorBody` type

**Step 1: Regenerate**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./...
```

**Step 2: Verify build + tests**

```bash
cd bin-api-manager && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS. If `gen.go` no longer includes a `Domain` field on `ErrorBody`, any code that referenced it (search: `grep -rn "ErrorBody" bin-api-manager --include="*.go" | grep -v gens/`) must be updated. Likely none — the generated `ErrorBody` was descriptive only.

---

### Task 6.3: Commit Phase 6

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
git add bin-openapi-manager/openapi/openapi.yaml bin-openapi-manager/gens/ bin-openapi-manager/go.mod bin-openapi-manager/go.sum bin-api-manager/gens/ bin-api-manager/go.mod bin-api-manager/go.sum
git status --short  # confirm what's staged
git commit -m "$(cat <<'EOF'
NOJIRA-remove-domain-from-external-error

- bin-openapi-manager: Drop domain property from ErrorBody schema and remove it from the required field list
- bin-openapi-manager: Regenerate openapi types
- bin-api-manager: Regenerate openapi server bindings to match updated ErrorBody schema
EOF
)"
```

---

## Phase 7 — RST documentation

### Task 7.1: Update `restful_api.rst`

**Files:**
- Modify: `bin-api-manager/docsdev/source/restful_api.rst`

**Step 1: Drop `domain` mention from the envelope description (around line 108)**

Find the sentence:
> Every 4xx/5xx response from the VoIPbin API ... contains a JSON error envelope with a canonical ``status``, a specific ``reason``, the originating ``domain``, a human-readable ``message``, and a ``request_id`` for support correlation. Branch on ``error.reason`` for debugging; ``error.status`` maps 1:1 to the HTTP status code.

Change to:
> Every 4xx/5xx response from the VoIPbin API ... contains a JSON error envelope with a canonical ``status``, a specific ``reason``, a human-readable ``message``, and a ``request_id`` for support correlation. Branch on ``error.reason`` for debugging; ``error.status`` maps 1:1 to the HTTP status code.

**Step 2: Drop the `domain` row from the field-level table further down the page**

Search for a table row that documents `domain` as a field of the error envelope (look around line 129+). Delete that row.

**Step 3: Verify no other `domain` references in this file describe the external envelope**

```bash
grep -n "domain" bin-api-manager/docsdev/source/restful_api.rst
```

Inspect remaining matches; drop any that describe the envelope as a client-observable field. Leave any that reference internal architecture (unlikely in a public doc) or unrelated topics.

---

### Task 7.2: Restructure `restful_api_errors.rst`

**Files:**
- Modify: `bin-api-manager/docsdev/source/restful_api_errors.rst`

**Step 1: Read the existing file structure**

```bash
grep -n "^[A-Z].*$\|^=\|^-\|^~" bin-api-manager/docsdev/source/restful_api_errors.rst | head -60
```

This reveals current section headings. The current grouping is by service (call-manager domain, billing-manager domain, etc.).

**Step 2: Restructure as two top-level groups**

Replace the existing reason-catalogue body with:

**a) "Generic / Cross-cutting Reasons"** (first, with a brief intro that these apply across all endpoints), listing every reason currently in the api-manager section:
- `INTERNAL`, `INVALID_ARGUMENT`, `INVALID_JSON_BODY`, `INVALID_ID`, `REQUEST_TIMEOUT`, `REQUEST_CANCELED`, `SERVICE_UNAVAILABLE`, `RESOURCE_NOT_FOUND`, `STATE_INVALID`, `INSUFFICIENT_BALANCE`, `RATE_LIMIT_EXCEEDED`, `ACCOUNT_FROZEN`, `PERMISSION_DENIED`, `DIRECT_ACCESS_NOT_SUPPORTED`, `AUTHENTICATION_REQUIRED`, `ROUTE_NOT_FOUND`.

**b) "Resource-Prefixed Reasons"** grouped by reason prefix (intrinsic to the reason code itself):
- "Call Reasons" (`CALL_*`)
- "Flow / Activeflow Reasons" (`FLOW_*`, `ACTIVEFLOW_*`)
- "Recording Reasons" (`RECORDING_*`)
- "Number Reasons" (`NUMBER_*`, plus `IDENTITY_VERIFICATION_REQUIRED`)
- "Trunk / Provider Reasons" (`TRUNK_*`, `PROVIDER_*`, `PROVIDERCALL_*`)
- "Customer / Accesskey Reasons" (`CUSTOMER_*`, `ACCESSKEY_*`)
- …enumerate the rest by scanning the existing file for prefixes that actually appear.

**Step 3: Remove all "emitted today by X-manager (via `cerrors.NotFound("<service>", ...)`)" provenance prose**

Search for those phrasings and delete them. The reasoning is in the design doc §5.1: provenance is now internal-only by design and risks drifting.

**Step 4: Remove the introductory note that explains how `domain` surfaces via the typed-passthrough translator**

Search for the text describing typed-passthrough domain surfacing (around line 14 in the current file). Delete or replace it with a brief note that the envelope contains `status / reason / message / request_id`.

---

### Task 7.3: Clean rebuild HTML

**Step 1: Set up Sphinx environment if not already present**

If you don't have a `.venv_docs`:
```bash
cd bin-api-manager/docsdev
python3 -m venv .venv_docs
source .venv_docs/bin/activate
pip install sphinx sphinx-rtd-theme sphinx-wagtail-theme sphinxcontrib-youtube
```

If it exists:
```bash
cd bin-api-manager/docsdev
source .venv_docs/bin/activate
```

**Step 2: Clean rebuild**

```bash
cd bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

Expected: build succeeds with no warnings about broken `:ref:` targets. If warnings appear about renamed sections, fix the cross-references in source files and rebuild.

**Step 3: Spot-check the rendered output**

```bash
ls build/html/restful_api.html build/html/restful_api_errors.html
```

Expected: both files exist. Optionally open them and visually confirm the new structure.

---

### Task 7.4: Commit Phase 7

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
git add bin-api-manager/docsdev/source/restful_api.rst bin-api-manager/docsdev/source/restful_api_errors.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "$(cat <<'EOF'
NOJIRA-remove-domain-from-external-error

- bin-api-manager: Drop "domain" from restful_api.rst envelope description and field table
- bin-api-manager: Restructure restful_api_errors.rst into Generic / Cross-cutting Reasons (first) and Resource-Prefixed Reasons (grouped by reason-code prefix); remove all "emitted by X-manager" provenance prose
- bin-api-manager: Clean rebuild docsdev/build/ HTML to match updated RST sources
EOF
)"
```

---

## Phase 8 — Final verification

### Task 8.1: Run full bin-api-manager verification workflow

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS at every step.

### Task 8.2: Run full bin-openapi-manager verification workflow

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS.

### Task 8.3: Re-run the CI grep guard

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
make lint-error-envelope
```

Expected: `OK: no open-coded "domain": literals in api-manager error sites.`

### Task 8.4: Sweep for any lingering `domain` test assertions across bin-api-manager

```bash
grep -rn '"domain"' bin-api-manager --include="*_test.go" | grep -v "MUST be absent\|absent from external\|key MUST"
```

Expected: empty (or only the absence-of-domain assertions added in this PR).

### Task 8.5: Commit any lingering vendor/mod/sum updates from Task 8.1–8.2 if not already committed

```bash
git status --short
# If anything is unstaged from go mod tidy / go generate, stage and commit:
git add -u
git commit -m "$(cat <<'EOF'
NOJIRA-remove-domain-from-external-error

- bin-api-manager: Sync go.mod, go.sum, and generated types after final verification
- bin-openapi-manager: Sync go.mod, go.sum after final verification
EOF
)"
```

(If nothing new is staged, skip this commit.)

---

## Phase 9 — Pre-PR sync with main

### Task 9.1: Fetch latest main and check for conflicts

From the worktree root:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-remove-domain-from-external-error
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
git log --oneline HEAD..origin/main
```

If conflicts exist, rebase or merge main, resolve, and re-run Phase 8 verification before continuing.

### Task 9.2: Stop here — DO NOT push or open a PR

The user must explicitly authorize push and PR creation. When they do, the PR title MUST be `NOJIRA-remove-domain-from-external-error` (matches branch) and the PR body MUST list affected projects with `bin-<service>:` prefixes (no AI attribution). Use squash merge only.

Suggested PR body:

```
Stop emitting the internal `domain` field from external HTTP error JSON responses returned by bin-api-manager. The field is preserved in the VoipbinError struct, internal RPC, and server-side logs, but no longer crosses the public API boundary.

- bin-api-manager: Add lib/apierror.EnvelopeFor as the single chokepoint that builds the external HTTP error envelope without "domain"
- bin-api-manager: Refactor 4 open-coded envelope sites (server/error.go, two in lib/middleware/authenticate.go, lib/middleware/ratelimit.go) to route through EnvelopeFor
- bin-api-manager: Update assertErrorResponse helper signature (drop wantDomain) and 97 call sites; add absence-of-domain assertions to envelope tests
- bin-api-manager: Add integration tests covering typed-error, nil-fallback, details-preserved, and panic-recovery (current-behavior) paths
- bin-api-manager: Drop "domain" from restful_api.rst envelope description; restructure restful_api_errors.rst by reason-code prefix (Generic / Cross-cutting first, then Resource-Prefixed)
- bin-openapi-manager: Drop "domain" property from ErrorBody schema and required field list; regenerate types
- monorepo: Add scripts/check-error-envelope.sh CI guard and Makefile lint-error-envelope target to prevent re-leaks under bin-api-manager/server/, lib/middleware/, and lib/service/
```

---

## Out-of-scope follow-ups (do NOT implement here; tracked in design §3.2)

- Replace `gin.Default()` in `bin-api-manager/cmd/api-manager/main.go` with `gin.New()` + a custom `RecoveryWithWriter` that calls `apierror.EnvelopeFor(nil, requestID)` so panic-recovery emits the canonical envelope instead of an empty body.
- Convert `bin-api-manager/lib/service/{auth,signup,boot,unregister}.go` `c.AbortWithStatus(400)` sites to construct `cerrors.InvalidArgument(...)` and route through `apierror.EnvelopeFor`. The grep guard already covers `lib/service/` so any conversion that re-introduces `"domain"` is caught.
- Add a custom `go/analysis` analyzer or `ruleguard` rule that flags any `c.JSON | c.AbortWithStatusJSON | c.IndentedJSON | c.SecureJSON | c.JSONP` call whose 2nd argument's static type is `*cerrors.VoipbinError` (catches direct-serialization bypasses that the regex grep cannot reliably match).
