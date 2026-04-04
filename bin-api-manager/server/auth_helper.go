package server

import (
	"monorepo/bin-api-manager/models/auth"

	"github.com/gin-gonic/gin"
)

// getAuthIdentity extracts the AuthIdentity from the gin context.
// Returns nil and false if the identity is not present.
func getAuthIdentity(c *gin.Context) (*auth.AuthIdentity, bool) {
	tmp, exists := c.Get("auth_identity")
	if !exists {
		return nil, false
	}
	a, ok := tmp.(*auth.AuthIdentity)
	if !ok {
		return nil, false
	}
	return a, true
}
