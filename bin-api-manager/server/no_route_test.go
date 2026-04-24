package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

func TestNoRouteEmitsEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.NoRoute(NoRoute())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1.0/definitely-not-a-route", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusNotFound, "ROUTE_NOT_FOUND", commonoutline.ServiceNameAPIManager)
}

func TestNoRouteUnknownMethod(t *testing.T) {
	// Gin's NoRoute fires for any HTTP method that lands on an unmatched path.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.NoRoute(NoRoute())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1.0/definitely-not-a-route", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d want 404", w.Code)
	}
}
