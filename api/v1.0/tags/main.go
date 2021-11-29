package tags

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/tags")

	targets.POST("", tagsPOST)
	targets.GET("", tagsGET)

	targets.DELETE("/:id", tagsIDDelete)
	targets.GET("/:id", tagsIDGet)
	targets.PUT("/:id", tagsIDPUT)
}
