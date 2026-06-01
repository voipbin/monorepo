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
