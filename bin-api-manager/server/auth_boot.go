package server

import (
	"github.com/gin-gonic/gin"
)

// PostAuthBoot is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/boot
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthBoot(c *gin.Context) {
	c.AbortWithStatus(404)
}
