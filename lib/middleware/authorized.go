package middleware

import "github.com/gin-gonic/gin"

// Authorized checks the request has authorized.
// If not, return 401
func Authorized(c *gin.Context) {
	_, exists := c.Get("customer")
	if !exists {
		c.AbortWithStatus(401)
		return
	}
}
