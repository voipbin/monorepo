package apiv1

import (
	"github.com/gin-gonic/gin"
	"gitlab.com/voipbin/bin-manager/api-manager/api/v1.0/auth"
	"gitlab.com/voipbin/bin-manager/api-manager/api/v1.0/conferences"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1.0")

	// v1.0
	auth.ApplyRoutes(v1)
	conferences.ApplyRoutes(v1)
}
