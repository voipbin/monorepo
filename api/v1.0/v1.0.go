package apiv1

import (
	"github.com/gin-gonic/gin"

	"gitlab.com/voipbin/bin-manager/api-manager/api/v1.0/calls"
	"gitlab.com/voipbin/bin-manager/api-manager/api/v1.0/conferences"
	"gitlab.com/voipbin/bin-manager/api-manager/api/v1.0/users"
	"gitlab.com/voipbin/bin-manager/api-manager/lib/middleware"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1.0", middleware.Authorized)

	// v1.0
	conferences.ApplyRoutes(v1)
	users.ApplyRoutes(v1)
	calls.ApplyRoutes(v1)
}
