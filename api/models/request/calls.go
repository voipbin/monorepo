package request

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/call"
)

// BodyCallsPOST is rquest body define for POST /calls
type BodyCallsPOST struct {
	Source      call.Address `json:"source" binding:"required"`
	Destination call.Address `json:"destination" binding:"required"`
	WebhookURI  string           `json:"webhook_uri"`
	Actions     []action.Action  `json:"actions"`
	// MachineDetection string          `json:"machine_detection"`
}

// ParamCallsGET is rquest param define for GET /calls
type ParamCallsGET struct {
	Pagination
}
