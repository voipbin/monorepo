package calls

import (
	"github.com/gin-gonic/gin"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/call"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	calls := r.Group("/calls")

	calls.POST("", callsPOST)
	calls.GET("/:id", callsIDDelete)
	// calls.DELETE("/:id", conferencesIDDELETE)
}

// RequestBodyCallsPOST is rquest body define for POST /calls
type RequestBodyCallsPOST struct {
	Source      call.Address    `json:"source" binding:"required"`
	Destination call.Address    `json:"destination" binding:"required"`
	Actions     []action.Action `json:"actions"`
	EventURL    string          `json:"event_url"`
	// MachineDetection string          `json:"machine_detection"`
}
