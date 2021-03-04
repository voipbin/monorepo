package request

import (
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

// V1DataCallsPost is
// v1 data type request struct for
// /v1/calls POST
type V1DataCallsPost struct {
	FlowID      uuid.UUID    `json:"flow_id"`
	UserID      uint64       `json:"user_id"`
	Source      call.Address `json:"source"`
	Destination call.Address `json:"destination"`
}

// V1DataCallsIDPost is
// v1 data type request struct for
// /v1/calls/<id> POST
type V1DataCallsIDPost struct {
	FlowID      uuid.UUID    `json:"flow_id"`
	UserID      uint64       `json:"user_id"`
	Source      call.Address `json:"source"`
	Destination call.Address `json:"destination"`
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

// V1DataCallsIDChainedCallIDs is
// v1 data type for V1DataCallsIDChainedCallIDs
// /v1/calls/<id>/chained-call-ids POST
type V1DataCallsIDChainedCallIDs struct {
	ChainedCallID uuid.UUID `json:"chained_call_id"`
}
