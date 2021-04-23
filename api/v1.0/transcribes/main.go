package transcribes

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin engine
func ApplyRoutes(r *gin.RouterGroup) {
	transcribes := r.Group("/transcribes")

	transcribes.POST("", transcribesPOST)
}
