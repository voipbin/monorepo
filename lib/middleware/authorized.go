package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Authorized checks the request has authorized.
// If not, return 401
func Authorized(c *gin.Context) {
	logrus.Debugf("Auth info: %v", c.Keys)

	_, exists := c.Get("agent")
	if !exists {
		c.AbortWithStatus(401)
		return
	}
}
