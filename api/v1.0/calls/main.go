package calls

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	calls := r.Group("/calls")

	calls.GET("", callsGET)
	calls.POST("", callsPOST)
	calls.DELETE("/:id", callsIDDelete)
	calls.GET("/:id", callsIDGET)
	calls.POST("/:id/hangup", callsIDHangupPOST)
}
