// Integration tests for the apierror package. These tests drive a real
// gin handler chain end-to-end and assert that EnvelopeFor's contract
// (no "domain" key in the response body) holds when the helper is
// invoked through c.AbortWithStatusJSON.
//
// Scope note: these tests intentionally exercise the apierror layer's
// gin serialization, NOT the server.abortWithError wrapper that wraps
// the helper in production. End-to-end coverage of abortWithError lives
// in bin-api-manager/server/error_test.go (TestAbortWithErrorSetsStatusAndBody,
// TestAbortWithErrorNilFallsBackToInternal, TestAbortWithErrorIncludesDetails),
// which exercises the wrapper through a gin chain and includes the
// structural domain-absence check. The two layers are tested independently
// so a regression in either surfaces with a precise failure signature.
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
	// Current behavior assertion: empty body on panic.
	//
	// gin.Recovery() today swallows panics and emits an empty 500 response.
	// If a future Recovery middleware (e.g., the follow-up tracked in
	// design doc §3.2 for envelope-on-panic) starts emitting a JSON body,
	// this assertion fires loudly so a human MUST update both the
	// expectation and the contract verification (no "domain" key in the
	// new body shape).
	if w.Body.Len() != 0 {
		// Defense in depth: even though the test is failing, also verify
		// the new body shape doesn't leak "domain" so the regression
		// signature is precise.
		assertNoDomainInBody(t, w.Body.String())
		t.Errorf("gin.Recovery now emits a body (%q); update TestIntegration_PanicRecovery_CurrentBehavior to assert the new envelope shape and add a "+
			"no-domain check at the wrapper layer.", w.Body.String())
	}
}
