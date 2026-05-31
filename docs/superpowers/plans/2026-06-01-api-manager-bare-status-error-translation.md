# api-manager bare-status error translation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `bin-api-manager` translate bare RPC status sentinels (404/409/429/etc.) to the correct client HTTP status instead of collapsing them to 500, fixing [#953](https://github.com/voipbin/monorepo/issues/953).

**Architecture:** Single-file change to the central error translator `translateToVoipbinError` in `bin-api-manager/server/error_translate.go`. Add a closed set of `requesthandler.Err*` → `cerrors` constructor cases (step 2, the sentinel-match section), mirroring `cerrors.HTTPStatusFor` in reverse and reusing the exact reason/message strings already used by the existing typed-sentinel cases. No backend changes; typed `VoipbinError` envelopes still take precedence via step 1.

**Tech Stack:** Go, `go test` (table-driven), gin + httptest (edge tests), `golangci-lint`.

**Spec:** `docs/superpowers/specs/2026-06-01-api-manager-bare-status-error-translation-design.md`

---

## Background (read before starting)

When a backend manager (e.g. `bin-ai-manager`) returns a **bare** status code via `simpleResponse(404)` — i.e. no typed `VoipbinError` body — the `requesthandler.parseResponse` layer converts it to a sentinel error from `HttpStatusErrorMap` (e.g. `requesthandler.ErrNotFound`). That sentinel propagates up to `bin-api-manager`'s `translateToVoipbinError`, which currently handles only `requesthandler.ErrBadRequest` (bare 400). Every other bare status falls through to the Default branch → `Internal` → HTTP 500.

This plan adds the missing cases. The mapping mirrors `cerrors.HTTPStatusFor` (`bin-common-handler/models/errors/rpc.go:54`) so a bare backend status round-trips back to the same client HTTP code:

| `requesthandler` sentinel | HTTP | `cerrors` constructor | Status | reason | message | wraps cause? |
|---|---|---|---|---|---|---|
| `ErrBadRequest` | 400 | `InvalidArgument` | `INVALID_ARGUMENT` | `INVALID_ARGUMENT` | "The request contains invalid data." | *(already present)* |
| `ErrUnauthorized` | 401 | `Unauthenticated` | `UNAUTHENTICATED` | `AUTHENTICATION_REQUIRED` | "Authentication is required." | no |
| `ErrPaymentRequired` | 402 | `PaymentRequired` | `PAYMENT_REQUIRED` | `INSUFFICIENT_BALANCE` | "Customer balance is below the minimum required for this operation." | yes |
| `ErrForbidden` | 403 | `PermissionDenied` | `PERMISSION_DENIED` | `PERMISSION_DENIED` | "You do not have permission to access this resource." | no |
| `ErrNotFound` | 404 | `NotFound` | `NOT_FOUND` | `RESOURCE_NOT_FOUND` | "The requested resource was not found." | no |
| `ErrConflict` | 409 | `FailedPrecondition` | `FAILED_PRECONDITION` | `STATE_INVALID` | "The operation is invalid for the current resource state." | yes |
| `ErrTooManyRequests` | 429 | `ResourceExhausted` | `RESOURCE_EXHAUSTED` | `RATE_LIMIT_EXCEEDED` | "Too many requests. Please retry later." | yes |
| `ErrServiceUnavailable` | 503 | `Unavailable` | `UNAVAILABLE` | `SERVICE_UNAVAILABLE` | "An upstream service is temporarily unavailable." | yes |
| `ErrInternal` | 500 | `Internal` | `INTERNAL` | `INTERNAL` | "An internal error occurred." | yes |

The "wraps cause?" column matches the wrap behavior of each existing typed-sentinel analogue in `error_translate.go` (e.g. the existing `INSUFFICIENT_BALANCE`/`STATE_INVALID`/`UNAVAILABLE`/`INTERNAL` cases call `.Wrap(err)`; the existing `PERMISSION_DENIED`/`RESOURCE_NOT_FOUND` cases do not). `.Wrap(err)` only sets the internal `Cause` chain (used by `errors.Is`); it is **stripped from the external client envelope** by `lib/apierror`, so it never changes the response body — it is purely about preserving the internal error chain.

**Already-imported, no new imports in the implementation file:** `error_translate.go` already imports `requesthandler` (the existing `ErrBadRequest` case uses it), `cerrors`, `commonoutline`, and `stderrors`. The two **test** files need import additions (called out per task).

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `bin-api-manager/server/error_translate.go` | Central servicehandler-error → `*VoipbinError` translator | Add 8 bare-status sentinel cases in step 2; update the stale doc comment. |
| `bin-api-manager/server/error_translate_test.go` | Unit tests for `translateToVoipbinError` | Add table test for all 9 bare-status sentinels + a `pkg/errors`-wrapped 404 variant. |
| `bin-api-manager/server/error_test.go` | Edge tests driving real gin recorder through `abortWithServiceError` | Add one HTTP round-trip test: wrapped bare-404 → `w.Code == 404`. |

---

## Task 1: Add bare-status sentinel cases to the translator (unit-tested)

**Files:**
- Modify: `bin-api-manager/server/error_translate.go` (function `translateToVoipbinError`, the step-2 switch; and the doc comment above it)
- Test: `bin-api-manager/server/error_translate_test.go`

- [ ] **Step 1: Write the failing unit test**

Add the `requesthandler` import to the test file's import block. The current import block (`error_translate_test.go:3-13`) is:

```go
import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"

	pkgerrors "github.com/pkg/errors"
)
```

Replace it with (adds one line — the `requesthandler` import):

```go
import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"

	pkgerrors "github.com/pkg/errors"
)
```

Then append these two test functions to the end of `error_translate_test.go`:

```go
// TestTranslateBareStatusSentinels verifies that a bare requesthandler
// HTTP-status sentinel (produced when a backend returns a bare
// simpleResponse(<code>) with no typed VoipbinError body) is translated to
// the matching cerrors status/reason, mirroring cerrors.HTTPStatusFor.
// This is the regression guard for issue #953.
func TestTranslateBareStatusSentinels(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus cerrors.Status
		wantReason string
	}{
		{"bad_request", requesthandler.ErrBadRequest, cerrors.StatusInvalidArgument, "INVALID_ARGUMENT"},
		{"unauthorized", requesthandler.ErrUnauthorized, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED"},
		{"payment_required", requesthandler.ErrPaymentRequired, cerrors.StatusPaymentRequired, "INSUFFICIENT_BALANCE"},
		{"forbidden", requesthandler.ErrForbidden, cerrors.StatusPermissionDenied, "PERMISSION_DENIED"},
		{"not_found", requesthandler.ErrNotFound, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND"},
		{"conflict", requesthandler.ErrConflict, cerrors.StatusFailedPrecondition, "STATE_INVALID"},
		{"too_many_requests", requesthandler.ErrTooManyRequests, cerrors.StatusResourceExhausted, "RATE_LIMIT_EXCEEDED"},
		{"service_unavailable", requesthandler.ErrServiceUnavailable, cerrors.StatusUnavailable, "SERVICE_UNAVAILABLE"},
		{"internal", requesthandler.ErrInternal, cerrors.StatusInternal, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateToVoipbinError(tt.err)
			if got.Status != tt.wantStatus || got.Reason != tt.wantReason {
				t.Errorf("got status=%q reason=%q want %q/%q", got.Status, got.Reason, tt.wantStatus, tt.wantReason)
			}
			// HTTPStatusFor must round-trip the resulting status back to the
			// HTTP code the backend originally emitted.
			if cerrors.HTTPStatusFor(got.Status) == 500 && tt.wantStatus != cerrors.StatusInternal {
				t.Errorf("status %q unexpectedly maps to HTTP 500", got.Status)
			}
		})
	}
}

// TestTranslateBareNotFoundWrapped verifies the production wrapping path:
// servicehandler wraps the bare-404 sentinel with pkg/errors.Wrapf (e.g.
// serviceHandler.aipromptproposalGet does errors.Wrapf(err, "could not get
// ai prompt proposal info")). The sentinel must still be recovered through
// the wrap.
func TestTranslateBareNotFoundWrapped(t *testing.T) {
	wrapped := pkgerrors.Wrapf(requesthandler.ErrNotFound, "could not get ai prompt proposal info")
	got := translateToVoipbinError(wrapped)
	if got.Status != cerrors.StatusNotFound {
		t.Errorf("wrapped bare 404 should map to NOT_FOUND, got %q", got.Status)
	}
	if got.Reason != "RESOURCE_NOT_FOUND" {
		t.Errorf("reason = %q want RESOURCE_NOT_FOUND", got.Reason)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd bin-api-manager && go test ./server/ -run 'TestTranslateBareStatus|TestTranslateBareNotFoundWrapped' -v`

Expected: FAIL. Cases like `unauthorized`, `not_found`, `conflict`, `too_many_requests`, `service_unavailable` currently fall through to the Default branch, so `got.Status` is `INTERNAL` / `got.Reason` is `INTERNAL` instead of the expected values. (`bad_request` already passes; `internal` already passes via Default — the others fail.)

- [ ] **Step 3: Write the minimal implementation**

In `error_translate.go`, find the existing `ErrBadRequest` case in the step-2 switch:

```go
		case stderrors.Is(err, requesthandler.ErrBadRequest):
			return cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "The request contains invalid data.")
```

Replace that single case with the full closed set (the `ErrBadRequest` case is unchanged; the 8 new cases are inserted immediately after it):

```go
		case stderrors.Is(err, requesthandler.ErrBadRequest):
			return cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "The request contains invalid data.")
		case stderrors.Is(err, requesthandler.ErrUnauthorized):
			return cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required.")
		case stderrors.Is(err, requesthandler.ErrPaymentRequired):
			return cerrors.PaymentRequired(commonoutline.ServiceNameAPIManager, "INSUFFICIENT_BALANCE",
				"Customer balance is below the minimum required for this operation.").Wrap(err)
		case stderrors.Is(err, requesthandler.ErrForbidden):
			return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "PERMISSION_DENIED", "You do not have permission to access this resource.")
		case stderrors.Is(err, requesthandler.ErrNotFound):
			return cerrors.NotFound(commonoutline.ServiceNameAPIManager, "RESOURCE_NOT_FOUND", "The requested resource was not found.")
		case stderrors.Is(err, requesthandler.ErrConflict):
			return cerrors.FailedPrecondition(commonoutline.ServiceNameAPIManager, "STATE_INVALID",
				"The operation is invalid for the current resource state.").Wrap(err)
		case stderrors.Is(err, requesthandler.ErrTooManyRequests):
			return cerrors.ResourceExhausted(commonoutline.ServiceNameAPIManager, "RATE_LIMIT_EXCEEDED",
				"Too many requests. Please retry later.").Wrap(err)
		case stderrors.Is(err, requesthandler.ErrServiceUnavailable):
			return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "SERVICE_UNAVAILABLE",
				"An upstream service is temporarily unavailable.").Wrap(err)
		case stderrors.Is(err, requesthandler.ErrInternal):
			return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err)
```

Then update the doc comment above `translateToVoipbinError` so it documents the full bare-status set instead of only the bare-400 exception. Find the existing comment lines (the priority-order block and the "Exception" paragraph):

```go
// translateToVoipbinError maps any error returned from a servicehandler
// into a *VoipbinError. Priority order:
//  1. Typed passthrough (errors.As).
//  2. Sentinel match (errors.Is against serviceerrors.Err*).
//  3. Transport-failure detection (context.Canceled / DeadlineExceeded).
//  4. Default: Internal with the original error wrapped as Cause.
```

Replace the priority-order line `//  2.` so it reads:

```go
//  2. Sentinel match (errors.Is against serviceerrors.Err* and the bare
//     requesthandler.Err* HTTP-status sentinels).
```

And find the "Exception" paragraph:

```go
// Exception: backend services that return a bare status code (e.g.
// simpleResponse(400)) without a typed VoipbinError body produce a
// requesthandler.ErrBadRequest sentinel instead of a VoipbinError.
// That case is handled explicitly below.
```

Replace it with:

```go
// Bare status codes: backend services that return a bare status code
// (e.g. simpleResponse(404)) without a typed VoipbinError body produce a
// requesthandler.Err* sentinel (via HttpStatusErrorMap) instead of a
// VoipbinError. Step 2 maps the closed set of these sentinels
// (400/401/402/403/404/409/429/503/500) back to the canonical cerrors
// status, mirroring cerrors.HTTPStatusFor in reverse so the client sees
// the same HTTP code the backend emitted. Statuses outside that set fall
// through to the Default branch (INTERNAL).
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd bin-api-manager && go test ./server/ -run 'TestTranslateBareStatus|TestTranslateBareNotFoundWrapped' -v`

Expected: PASS (all sub-tests).

- [ ] **Step 5: Run the full server-package test to confirm no regression**

Run: `cd bin-api-manager && go test ./server/ -run TestTranslate -v`

Expected: PASS — including the pre-existing `TestTranslateSentinels`, `TestTranslateTypedPassthrough`, `TestTranslatePkgErrorsWrappedTypedError`, `TestTranslateLegacyStringFallsThroughToInternal`, `TestTranslateDefault`. (The "legacy string falls through to INTERNAL" test must still pass — bare *string* errors are not `requesthandler.Err*` sentinels, so they are unaffected.)

- [ ] **Step 6: Commit**

```bash
cd bin-api-manager
git add server/error_translate.go server/error_translate_test.go
git commit -m "NOJIRA-Fix-api-manager-bare-status-error-translation

- bin-api-manager: Translate bare requesthandler HTTP-status sentinels (401/402/403/404/409/429/503/500) to matching VoipbinError statuses in translateToVoipbinError; fixes 500-instead-of-404 for nonexistent aipromptproposals IDs (#953)"
```

---

## Task 2: Add edge-level HTTP round-trip test (boundary regression guard)

This test drives the real gin → `abortWithServiceError` → `abortWithError` → `HTTPStatusFor` path the client actually observes, asserting the wrapped bare-404 produces HTTP `404` with the correct body. The implementation already landed in Task 1, so this test passes immediately; it exists as a permanent boundary guard (the Task 1 unit test only checks the translator's `Status`/`Reason`, not the HTTP status code written to the response).

**Files:**
- Test: `bin-api-manager/server/error_test.go`

- [ ] **Step 1: Add the edge test**

Add two imports to `error_test.go`. The current import block (`error_test.go:3-16`) is:

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)
```

Replace it with (adds `requesthandler` and `pkgerrors`):

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/pkg/errors"
)
```

Then append this test function to the end of `error_test.go` (it reuses the existing `assertErrorResponse` helper, which asserts both `w.Code` via `HTTPStatusFor` and the response body's status/reason):

```go
// TestAbortWithServiceErrorBareNotFoundRoundTrips404 guards the full
// client-observed path for issue #953: a bare requesthandler.ErrNotFound
// (emitted when a backend returns simpleResponse(404)), wrapped by
// servicehandler with pkg/errors.Wrapf, must surface as HTTP 404 with the
// RESOURCE_NOT_FOUND envelope — not 500.
func TestAbortWithServiceErrorBareNotFoundRoundTrips404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithServiceError(c, pkgerrors.Wrapf(requesthandler.ErrNotFound, "could not get ai prompt proposal info"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assertErrorResponse(t, w, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND")
}
```

- [ ] **Step 2: Run the edge test**

Run: `cd bin-api-manager && go test ./server/ -run TestAbortWithServiceErrorBareNotFoundRoundTrips404 -v`

Expected: PASS — `w.Code == 404`, body `status == "NOT_FOUND"`, `reason == "RESOURCE_NOT_FOUND"`, `request_id` present, no `domain` key.

- [ ] **Step 3: Run the full error_test.go suite to confirm no regression**

Run: `cd bin-api-manager && go test ./server/ -run 'TestAbortWith|TestAssertErrorResponseHelper' -v`

Expected: PASS (all pre-existing edge tests plus the new one).

- [ ] **Step 4: Commit**

```bash
cd bin-api-manager
git add server/error_test.go
git commit -m "NOJIRA-Fix-api-manager-bare-status-error-translation

- bin-api-manager: Add edge-level HTTP round-trip test asserting wrapped bare-404 surfaces as 404 NOT_FOUND (#953)"
```

---

## Task 3: Full verification workflow

**Files:** none (verification only)

- [ ] **Step 1: Run the mandatory verification workflow**

Run:

```bash
cd bin-api-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all five steps succeed. `go test ./...` runs the whole `bin-api-manager` suite (including `server/`). `golangci-lint` reports no new issues in `server/error_translate.go` / test files.

Notes:
- No new third-party dependency is added (`requesthandler`, `cerrors`, `pkg/errors` are all already in `go.mod`), so `go mod tidy` should produce no changes. If it does modify `go.mod`/`go.sum`, that is expected housekeeping — stage and commit those files too.
- `vendor/` is **not** committed (root `.gitignore` excludes it). Do **not** `git add -f vendor/`.

- [ ] **Step 2: Commit any go.mod/go.sum changes (only if the workflow modified them)**

```bash
cd bin-api-manager
git status --short go.mod go.sum
# If either file changed:
git add go.mod go.sum
git commit -m "NOJIRA-Fix-api-manager-bare-status-error-translation

- bin-api-manager: go mod tidy housekeeping"
```

If `git status` shows no changes to `go.mod`/`go.sum`, skip this commit.

---

## Out of scope (do NOT do in this plan)

- **No `bin-ai-manager` or `bin-common-handler` changes.** The fix is entirely within `bin-api-manager/server/`.
- **Do not touch** the "proposal not found" → 404 branches in ai-manager's accept/reject listenhandlers. They are redundant for the common api-manager path but still fire on a TOCTOU race or a direct/non-api-manager RPC caller — not dead code, not removable.
- **api-validator follow-up (separate repo, separate PR):** after this fix is deployed, un-`xfail` the 4 affected tests in `monorepo-monitoring/api-validator`. Track separately; not part of this monorepo PR.

## Done criteria

- `GET`, `DELETE`, `POST .../accept`, `POST .../reject` on a nonexistent `aipromptproposals` ID return `404 NOT_FOUND` (verified at the translator + edge-test level).
- Bare backend statuses 401/402/403/404/409/429/503/500 translate to the matching client HTTP status.
- Typed `VoipbinError` precedence unchanged; legacy bare-string errors still degrade to INTERNAL.
- Full verification workflow passes in `bin-api-manager`.
