package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
)

// PostAuthUnregister is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/unregister
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthUnregister(c *gin.Context, params openapi_server.PostAuthUnregisterParams) {
	c.AbortWithStatus(404)
}

// DeleteAuthUnregister is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/unregister
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) DeleteAuthUnregister(c *gin.Context, params openapi_server.DeleteAuthUnregisterParams) {
	c.AbortWithStatus(404)
}
