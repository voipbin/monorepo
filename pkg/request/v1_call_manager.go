package request

import (
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
)

// V1DataAsterisksIDChannelsIDHealth is
// v1 data type request struct for
// AsterisksIDChannelsIDHealth
// /v1/asterisks/<id>/channels/<id>/health-check POST
type V1DataAsterisksIDChannelsIDHealth struct {
	RetryCount    int `json:"retry_count"`
	RetryCountMax int `json:"retry_count_max"`
	Delay         int `json:"delay"`
}

// V1DataCallsIDHealth is
// v1 data type request struct for
// CallsIDHealth
// /v1/calls/<id>/health-check POST
type V1DataCallsIDHealth struct {
	RetryCount int `json:"retry_count"`
	Delay      int `json:"delay"`
}

// V1DataCallsIDActionTimeout is
// v1 data type for CallsIDActionTimeout
// /v1/calls/<id>/action-timeout POST
type V1DataCallsIDActionTimeout struct {
	ActionID   uuid.UUID   `json:"action_id"`
	ActionType action.Type `json:"action_type"`
	TMExecute  string      `json:"tm_execute"` // represent when this action has executed.
}

// V1DataConferencesIDDelete is
// v1 data type request struct for
// /v1/conferences/<id>" DELETE
type V1DataConferencesIDDelete struct {
	Reason string `json:"reason,omitempty"`
}
