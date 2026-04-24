package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
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

// assertErrorResponse is a test helper shared across handler tests in
// subsequent migration PRs. It asserts the HTTP status code matches
// the canonical Status AND the response body's status/reason fields
// match the expected values.
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, wantStatus cerrors.Status, wantReason string) {
	t.Helper()
	if got, want := w.Code, cerrors.HTTPStatusFor(wantStatus); got != want {
		t.Errorf("status code = %d want %d", got, want)
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
