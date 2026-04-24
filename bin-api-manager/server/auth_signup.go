package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

// PostAuthSignup is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/signup
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthSignup(c *gin.Context) {
	abortWithError(c, cerrors.NotFound(
		commonoutline.ServiceNameAPIManager,
		"ROUTE_NOT_FOUND",
		"The requested endpoint does not exist on this path; use /auth/signup (no /v1.0 prefix).",
	))
}

// PostAuthEmailVerify is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/email-verify
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthEmailVerify(c *gin.Context) {
	abortWithError(c, cerrors.NotFound(
		commonoutline.ServiceNameAPIManager,
		"ROUTE_NOT_FOUND",
		"The requested endpoint does not exist on this path; use /auth/email-verify (no /v1.0 prefix).",
	))
}
