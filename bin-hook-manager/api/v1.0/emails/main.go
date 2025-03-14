package emails

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	g := r.Group("/emails")

	g.POST("/:target", emailsPOST)
}
