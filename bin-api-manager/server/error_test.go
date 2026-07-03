package server

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

func TestAbortWithErrorSetsStatusAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithError(c, cerrors.NotFound(commonoutline.ServiceNameCallManager, "CALL_NOT_FOUND", "The call was not found."))
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
	if body.Error.Message != "The call was not found." {
		t.Errorf("wrong message: %q", body.Error.Message)
	}
	if body.Error.RequestID == "" {
		t.Error("request_id missing from response body")
	}
	// Structural check: parse the body and verify the "domain" key is
	// absent from the error object (not a substring scan, which would
	// false-positive on a Details payload containing a field named
	// "domain").
	var fullBody map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &fullBody); err != nil {
		t.Fatalf("unmarshal full body for domain check: %v; body=%s", err, w.Body.String())
	}
	errObj, ok := fullBody["error"].(map[string]any)
	if !ok {
		t.Fatalf("body.error is not an object: %+v", fullBody)
	}
	if _, hasDomain := errObj["domain"]; hasDomain {
		t.Errorf("domain key MUST be absent from external response; body=%s", w.Body.String())
	}
}

func TestAbortWithErrorNilFallsBackToInternal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithError(c, nil)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d want 500", w.Code)
	}
}

func TestAbortWithServiceErrorHandlesTypedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithServiceError(c, cerrors.PermissionDenied(commonoutline.ServiceNameBillingManager, "BILLING_ACCESS_DENIED", "Not allowed."))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d want 403", w.Code)
	}
}

func TestAbortWithServiceErrorDefaultsToInternal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithServiceError(c, fmt.Errorf("some unknown error"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d want 500", w.Code)
	}
}

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

func TestAbortWithErrorIncludesDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		e := cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "FIELD_VIOLATION", "Validation failed.")
		e.Details = []map[string]any{
			{"field": "phone", "issue": "invalid E.164"},
		}
		abortWithError(c, e)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	var body struct {
		Error struct {
			Details []map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Error.Details) != 1 {
		t.Fatalf("expected 1 details entry, got %+v", body.Error.Details)
	}
	if body.Error.Details[0]["field"] != "phone" {
		t.Errorf("details content wrong: %+v", body.Error.Details[0])
	}
}

func TestAbortWithErrorOmitsEmptyDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		abortWithError(c, cerrors.NotFound(commonoutline.ServiceNameCallManager, "CALL_NOT_FOUND", "x"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	// details must be absent when not set on the VoipbinError.
	if !strings.Contains(w.Body.String(), `"error"`) {
		t.Fatal("no error field")
	}
	if strings.Contains(w.Body.String(), `"details"`) {
		t.Errorf("details should be omitted when empty; body=%s", w.Body.String())
	}
}

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
	// Structural check: parse the body and verify the "domain" key is
	// absent from the error object (not a substring scan, which would
	// false-positive on a Details payload containing a field named
	// "domain").
	var full map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &full); err != nil {
		t.Fatalf("unmarshal full body for domain check: %v; body=%s", err, w.Body.String())
	}
	errObj, ok := full["error"].(map[string]any)
	if !ok {
		t.Fatalf("body.error is not an object: %+v", full)
	}
	if _, hasDomain := errObj["domain"]; hasDomain {
		t.Errorf("domain key MUST be absent from external response; body=%s", w.Body.String())
	}
}

// --- BindingErrorHandler (ETC-3) ---
//
// See docs/plans/2026-07-04-standardize-binding-error-envelope.md for
// the full design and review history. These tests specifically pin
// down the round-1/2/3/4 review findings as regressions:
//   - round 1: raw err.Error() must never reach the public message.
//   - round 2: hyphenated parameter names must not be truncated.
//   - round 3: "missing" vs "malformed" must be a distinct reason.
//   - round 4 (defensive, currently unreachable in production): a
//     statusCode other than 400 must fail safe to INTERNAL.

func newBindingErrorTestRouter(t *testing.T, err error, statusCode int) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		BindingErrorHandler(c, err, statusCode)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	return w
}

func TestBindingErrorHandler_InvalidFormat_UsesInvalidReason(t *testing.T) {
	w := newBindingErrorTestRouter(t, fmt.Errorf("Invalid format for parameter id: invalid UUID length"), http.StatusBadRequest)
	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_REQUEST_PARAMETER")

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if want := `The parameter "id" has an invalid format.`; body.Error.Message != want {
		t.Errorf("message = %q want %q", body.Error.Message, want)
	}
	// Round-1 regression guard: the raw oapi-codegen/Go internals must
	// never leak into the curated public message.
	if strings.Contains(body.Error.Message, "invalid UUID length") {
		t.Errorf("raw error text leaked into public message: %q", body.Error.Message)
	}
}

func TestBindingErrorHandler_HyphenatedParamName_ExtractsWhole(t *testing.T) {
	w := newBindingErrorTestRouter(t, fmt.Errorf("Invalid format for parameter billing-id: invalid UUID length"), http.StatusBadRequest)
	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_REQUEST_PARAMETER")

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Round-2 regression guard: a plain \w+ regex truncates at the
	// hyphen, producing "billing" instead of "billing-id".
	if want := `The parameter "billing-id" has an invalid format.`; body.Error.Message != want {
		t.Errorf("message = %q want %q (hyphenated name must not be truncated)", body.Error.Message, want)
	}
}

func TestBindingErrorHandler_MissingRequiredParam_UsesMissingReason(t *testing.T) {
	w := newBindingErrorTestRouter(t, fmt.Errorf("Query argument aicall_id is required, but not found"), http.StatusBadRequest)
	// Round-3 regression guard: missing must be its own reason, not
	// folded into INVALID_REQUEST_PARAMETER via message-text-only
	// differentiation.
	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "MISSING_REQUEST_PARAMETER")

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if want := `The parameter "aicall_id" is required.`; body.Error.Message != want {
		t.Errorf("message = %q want %q", body.Error.Message, want)
	}
}

func TestBindingErrorReason_PrefixOverlap_MalformedWins(t *testing.T) {
	// Pins down the prefix-exclusivity invariant classifyBindingError
	// relies on: even if a future oapi-codegen version's %w-wrapped
	// inner error text happened to contain "is required, but not
	// found", the malformed-value case must still win because its own
	// outer text always starts with "Invalid format for parameter ".
	inner := fmt.Errorf("value is required, but not found in enum")
	outer := fmt.Errorf("Invalid format for parameter id: %w", inner)

	reason, message := classifyBindingError(outer.Error())
	if reason != "INVALID_REQUEST_PARAMETER" {
		t.Errorf("reason = %q want INVALID_REQUEST_PARAMETER (malformed-value prefix must win over an accidental inner substring match)", reason)
	}
	if strings.Contains(message, "is required") {
		t.Errorf("message incorrectly took the 'missing' wording: %q", message)
	}
}

func TestBindingErrorHandler_UnparseableErrorText_FallsBackGeneric(t *testing.T) {
	w := newBindingErrorTestRouter(t, fmt.Errorf("something oapi-codegen has never said before"), http.StatusBadRequest)
	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_REQUEST_PARAMETER")

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if want := "One or more request parameters are invalid."; body.Error.Message != want {
		t.Errorf("message = %q want %q", body.Error.Message, want)
	}
}

func TestBindingErrorHandler_UnexpectedStatusCode_FallsBackToInternal(t *testing.T) {
	// Round-4 defensive guard: unreachable with today's oapi-codegen
	// (every real call site passes http.StatusBadRequest), but kept as
	// a regression guard against a future `go generate` that might
	// pass a different status code — must never be silently reported
	// as a 400 client error.
	w := newBindingErrorTestRouter(t, fmt.Errorf("Invalid format for parameter id: boom"), http.StatusInternalServerError)
	assertErrorResponse(t, w, cerrors.StatusInternal, "INTERNAL")
}
