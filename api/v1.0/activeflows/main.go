package activeflows

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	a := r.Group("/activeflows")

	a.POST("", activeflowsPOST)
	a.GET("", activeflowsGET)
	a.DELETE("/:id", activeflowsIDDELETE)
	a.GET("/:id", activeflowsIDGET)
	a.POST("/:id/stop", activeflowsIDStopPOST)
}
