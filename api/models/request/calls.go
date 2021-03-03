package request

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// BodyCallsPOST is rquest body define for POST /calls
type BodyCallsPOST struct {
	Source      models.CallAddress `json:"source" binding:"required"`
	Destination models.CallAddress `json:"destination" binding:"required"`
	Actions     []models.Action    `json:"actions"`
	EventURL    string             `json:"event_url"`
	// MachineDetection string          `json:"machine_detection"`
}

// ParamCallsGET is rquest param define for GET /calls
type ParamCallsGET struct {
	Pagination
}
