package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"

	"monorepo/bin-api-manager/gens/openapi_server"
)

// PostAuthPasswordForgot is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router (not under the /v1.0 prefix).
func (h *server) PostAuthPasswordForgot(c *gin.Context) {
	abortWithError(c, cerrors.NotFound(
		commonoutline.ServiceNameAPIManager,
		"ROUTE_NOT_FOUND",
		"Use /auth/password-forgot (no /v1.0 prefix).",
	))
}

// GetAuthPasswordReset is a stub to satisfy the generated ServerInterface.
func (h *server) GetAuthPasswordReset(c *gin.Context, params openapi_server.GetAuthPasswordResetParams) {
	abortWithError(c, cerrors.NotFound(
		commonoutline.ServiceNameAPIManager,
		"ROUTE_NOT_FOUND",
		"Use /auth/password-reset (no /v1.0 prefix).",
	))
}

// PostAuthPasswordReset is a stub to satisfy the generated ServerInterface.
func (h *server) PostAuthPasswordReset(c *gin.Context) {
	abortWithError(c, cerrors.NotFound(
		commonoutline.ServiceNameAPIManager,
		"ROUTE_NOT_FOUND",
		"Use /auth/password-reset (no /v1.0 prefix).",
	))
}
