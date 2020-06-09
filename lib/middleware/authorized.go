package middleware

import "github.com/gin-gonic/gin"

func Authorized(c *gin.Context) {
	_, exists := c.Get("user")
	if !exists {
		c.AbortWithStatus(401)
		return
	}
}
