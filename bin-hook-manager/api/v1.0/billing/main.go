package billing

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	g := r.Group("/billing")

	g.POST("/paddle", billingPaddlePOST)
}
