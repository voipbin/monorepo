package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

func TestAuthStubs_ReturnRouteNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		method  string
		path    string
		handler func(h *server, c *gin.Context)
	}{
		{"auth_boot", http.MethodPost, "/v1.0/auth/boot", func(h *server, c *gin.Context) { h.PostAuthBoot(c) }},
		{"auth_signup", http.MethodPost, "/v1.0/auth/signup", func(h *server, c *gin.Context) { h.PostAuthSignup(c) }},
		{"auth_email_verify", http.MethodPost, "/v1.0/auth/email-verify", func(h *server, c *gin.Context) { h.PostAuthEmailVerify(c) }},
		{"auth_unregister_post", http.MethodPost, "/v1.0/auth/unregister", func(h *server, c *gin.Context) { h.PostAuthUnregister(c, openapi_server.PostAuthUnregisterParams{}) }},
		{"auth_unregister_delete", http.MethodDelete, "/v1.0/auth/unregister", func(h *server, c *gin.Context) {
			h.DeleteAuthUnregister(c, openapi_server.DeleteAuthUnregisterParams{})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &server{}
			r := gin.New()
			r.Use(middleware.RequestID())
			r.Any(tt.path, func(c *gin.Context) {
				tt.handler(h, c)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			r.ServeHTTP(w, req)

			assertErrorResponse(t, w, cerrors.StatusNotFound, "ROUTE_NOT_FOUND", commonoutline.ServiceNameAPIManager)
		})
	}
}
