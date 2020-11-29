package apiv1

import (
	"github.com/gin-gonic/gin"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/api/v1.0/tts"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1.0")

	// v1.0
	tts.ApplyRoutes(v1)
}
