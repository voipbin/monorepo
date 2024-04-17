package ws

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/ws")

	targets.GET("", wsGET)
}
