package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

// PostAuthBoot is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/boot
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthBoot(c *gin.Context) {
	abortWithError(c, cerrors.NotFound(
		commonoutline.ServiceNameAPIManager,
		"ROUTE_NOT_FOUND",
		"The requested endpoint does not exist on this path; use /auth/boot (no /v1.0 prefix).",
	))
}
