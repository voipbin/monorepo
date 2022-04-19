package apiv1

import (
	"github.com/gin-gonic/gin"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/agents"
	availablenumbers "gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/available_numbers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/calls"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/conferences"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/customers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/domains"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/extensions"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/flows"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/messages"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/numbers"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/outdials"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/queues"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/recordingfiles"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/recordings"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/tags"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0/transcribes"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1.0", middleware.Authorized)

	// v1.0
	agents.ApplyRoutes(v1)
	availablenumbers.ApplyRoutes(v1)
	calls.ApplyRoutes(v1)
	conferences.ApplyRoutes(v1)
	customers.ApplyRoutes(v1)
	domains.ApplyRoutes(v1)
	extensions.ApplyRoutes(v1)
	flows.ApplyRoutes(v1)
	messages.ApplyRoutes(v1)
	numbers.ApplyRoutes(v1)
	outdials.ApplyRoutes(v1)
	queues.ApplyRoutes(v1)
	recordings.ApplyRoutes(v1)
	recordingfiles.ApplyRoutes(v1)
	tags.ApplyRoutes(v1)
	transcribes.ApplyRoutes(v1)
	// users.ApplyRoutes(v1)
}
