package transcribes

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin engine
func ApplyRoutes(r *gin.RouterGroup) {
	transcribes := r.Group("/transcribes")

	transcribes.POST("", transcribesPOST)
	transcribes.GET("", transcribesGET)

	transcribes.GET("/:id", transcribesIDGET)
	transcribes.DELETE("/:id", transcribesIDDelete)

	transcribes.POST("/:id/stop", transcribesIDStopPOST)
}
