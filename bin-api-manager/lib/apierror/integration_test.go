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
