package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
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

// V1DataCallsIDPost is
// v1 data type request struct for
// /v1/calls/<id> POST
type V1DataCallsIDPost struct {
	FlowID      uuid.UUID       `json:"flow_id"`
	UserID      uint64          `json:"user_id"`
	Source      address.Address `json:"source"`
	Destination address.Address `json:"destination"`
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

// V1DataConferencesIDPost is
// v1 data type request struct for
// /v1/conferences/<id>" POST
type V1DataConferencesIDPost struct {
	Type    conference.Type        `json:"type"`
	UserID  uint64                 `json:"user_id"`
	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Timeout int                    `json:"timeout"` // timeout. second
	Data    map[string]interface{} `json:"data"`
}

// V1DataCallsIDChainedCallIDsPost is
// v1 data type for V1DataCallsIDChainedCallIDs
// /v1/calls/<id>/chained-call-ids POST
type V1DataCallsIDChainedCallIDsPost struct {
	ChainedCallID uuid.UUID `json:"chained_call_id"`
}
