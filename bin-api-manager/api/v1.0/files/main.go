package files

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	files := r.Group("/files")

	files.GET("", filesGET)
	files.POST("", filesPOST)
	files.DELETE("/:id", filesIDDELETE)
	files.GET("/:id", filesIDGET)
}
