package apiv1

import (
	"github.com/gin-gonic/gin"
	"gitlab.com/voipbin/bin-manager/api-manager/api/v1.0/auth"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1.0")
	v1.GET("/ping", ping)

	// /auth
	auth.ApplyRoutes(v1)
}

func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
