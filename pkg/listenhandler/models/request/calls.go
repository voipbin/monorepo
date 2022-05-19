package request

import (
	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// V1DataCallsPost is
// v1 data type request struct for
// /v1/calls POST
type V1DataCallsPost struct {
	FlowID       uuid.UUID         `json:"flow_id"`
	CustomerID   uuid.UUID         `json:"customer_id"`
	MasterCallID uuid.UUID         `json:"master_call_id"`
	Source       address.Address   `json:"source"`
	Destinations []address.Address `json:"destinations"`
}

// V1DataCallsIDPost is
// v1 data type request struct for
// /v1/calls/<id> POST
type V1DataCallsIDPost struct {
	FlowID       uuid.UUID       `json:"flow_id"`
	ActiveflosID uuid.UUID       `json:"activeflow_id"`
	CustomerID   uuid.UUID       `json:"customer_id"`
	MasterCallID uuid.UUID       `json:"master_call_id"`
	Source       address.Address `json:"source"`
	Destination  address.Address `json:"destination"`
}

// V1DataCallsIDHealthPost is
// v1 data type request struct for
// CallsIDHealth
// /v1/calls/<id>/health-check POST
type V1DataCallsIDHealthPost struct {
	RetryCount int `json:"retry_count"`
	Delay      int `json:"delay"`
}

// V1DataCallsIDActionTimeoutPost is
// v1 data type for CallsIDActionTimeout
// /v1/calls/<id>/action-timeout POST
type V1DataCallsIDActionTimeoutPost struct {
	ActionID   uuid.UUID     `json:"action_id"`
	ActionType fmaction.Type `json:"action_type"`
	TMExecute  string        `json:"tm_execute"` // represent when this action has executed.
}

// V1DataCallsIDChainedCallIDsPost is
// v1 data type for V1DataCallsIDChainedCallIDsPost
// /v1/calls/<id>/chained-call-ids POST
type V1DataCallsIDChainedCallIDsPost struct {
	ChainedCallID uuid.UUID `json:"chained_call_id"`
}

// V1DataCallsIDActionNextPost is
// v1 data type for
// /v1/calls/<id>/action-next POST
type V1DataCallsIDActionNextPost struct {
	Force bool `json:"force"`
}

// V1DataCallsIDExternalMediaPost is
// v1 data type for V1DataCallsIDExternalMediaPost
// /v1/calls/<id>/external-media POST
type V1DataCallsIDExternalMediaPost struct {
	ExternalHost   string `json:"external_host"`
	Encapsulation  string `json:"encapsulation"`
	Transport      string `json:"transport"`
	ConnectionType string `json:"connection_type"`
	Format         string `json:"format"`
	Direction      string `json:"direction"`
}

// V1DataCallsIDDigitsPost is
// v1 data type for V1DataCallsIDDigitsPost
// /v1/calls/<id>/digits POST
type V1DataCallsIDDigitsPost struct {
	Digits string `json:"digits"`
}
