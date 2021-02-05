package flows

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	flows := r.Group("/flows")

	flows.GET("", flowsGET)
	flows.POST("", flowsPOST)

	flows.DELETE("/:id", flowsIDDELETE)
	flows.GET("/:id", flowsIDGET)
	flows.PUT("/:id", flowsIDPUT)
}
