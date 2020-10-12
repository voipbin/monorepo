package flows

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	flows := r.Group("/flows")

	flows.POST("", flowsPOST)
	flows.GET("", flowsGET)
	flows.GET("/:id", flowsIDGET)
}
