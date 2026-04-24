package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

// PostAuthUnregister is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/unregister
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthUnregister(c *gin.Context, params openapi_server.PostAuthUnregisterParams) {
	abortWithError(c, cerrors.NotFound(
		commonoutline.ServiceNameAPIManager,
		"ROUTE_NOT_FOUND",
		"The requested endpoint does not exist on this path; use /auth/unregister (no /v1.0 prefix).",
	))
}

// DeleteAuthUnregister is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/unregister
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) DeleteAuthUnregister(c *gin.Context, params openapi_server.DeleteAuthUnregisterParams) {
	abortWithError(c, cerrors.NotFound(
		commonoutline.ServiceNameAPIManager,
		"ROUTE_NOT_FOUND",
		"The requested endpoint does not exist on this path; use /auth/unregister (no /v1.0 prefix).",
	))
}
