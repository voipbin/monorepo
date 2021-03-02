package apiv1

import (
	"github.com/gin-gonic/gin"

	availablenumbers "gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/available_numbers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/calls"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/conferences"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/domains"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/extensions"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/flows"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/recordingfiles"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/recordings"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/users"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1.0", middleware.Authorized)

	// v1.0
	availablenumbers.ApplyRoutes(v1)
	calls.ApplyRoutes(v1)
	conferences.ApplyRoutes(v1)
	domains.ApplyRoutes(v1)
	extensions.ApplyRoutes(v1)
	flows.ApplyRoutes(v1)
	recordings.ApplyRoutes(v1)
	recordingfiles.ApplyRoutes(v1)
	users.ApplyRoutes(v1)
}
