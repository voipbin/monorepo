package queuecalls

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	queuecalls := r.Group("/queuecalls")

	queuecalls.GET("/:id", queuecallsIDGET)
	queuecalls.DELETE("/:id", queuecallsIDDELETE)
}
