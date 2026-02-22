package server

import (
	"github.com/gin-gonic/gin"
)

// PostAuthSignup is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/signup
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthSignup(c *gin.Context) {
	c.AbortWithStatus(404)
}

// PostAuthEmailVerify is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/email-verify
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthEmailVerify(c *gin.Context) {
	c.AbortWithStatus(404)
}

// PostAuthCompleteSignup is a stub to satisfy the generated ServerInterface.
// The actual handler is registered directly on the Gin router at /auth/complete-signup
// (not under the /v1.0 prefix), so this generated route is never called.
func (h *server) PostAuthCompleteSignup(c *gin.Context) {
	c.AbortWithStatus(404)
}
