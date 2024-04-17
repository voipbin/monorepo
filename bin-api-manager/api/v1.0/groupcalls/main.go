package groupcalls

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	flows := r.Group("/groupcalls")

	flows.GET("", groupcallsGET)
	flows.POST("", groupcallsPOST)

	flows.DELETE("/:id", groupcallsIDDELETE)
	flows.GET("/:id", groupcallsIDGET)

	flows.POST("/:id/hangup", groupcallsIDHangupPOST)
}
