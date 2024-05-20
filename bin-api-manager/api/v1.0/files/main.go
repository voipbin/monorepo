package files

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	files := r.Group("/files")

	// calls.GET("", callsGET)
	files.POST("", filesPOST)
	// calls.DELETE("/:id", callsIDDelete)
	// calls.GET("/:id", callsIDGET)
}
