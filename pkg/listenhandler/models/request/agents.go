package request

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// V1DataAgentsPost is
// v1 data type request struct for
// /v1/agents POST
type V1DataAgentsPost struct {
	CustomerID uuid.UUID               `json:"customer_id"`
	Username   string                  `json:"username"`
	Password   string                  `json:"password"`
	Name       string                  `json:"name"`
	Detail     string                  `json:"detail"`
	RingMethod string                  `json:"ring_method"`
	Permission uint64                  `json:"permission"`
	TagIDs     []uuid.UUID             `json:"tag_ids"`
	Addresses  []commonaddress.Address `json:"addresses"`
}

// V1DataAgentsUsernameLoginPost is
// v1 data type request struct for
// /v1/agents/<username>/login POST
type V1DataAgentsUsernameLoginPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Password   string    `json:"password"`
}

// V1DataAgentsIDPut is
// v1 data type request struct for
// /v1/agents/<agent-id> PUT
type V1DataAgentsIDPut struct {
	Name       string `json:"name"`
	Detail     string `json:"detail"`
	RingMethod string `json:"ring_method"`
}

// V1DataAgentsIDAddressesPut is
// v1 data type request struct for
// /v1/agents/<agent-id>/addresses PUT
type V1DataAgentsIDAddressesPut struct {
	Addresses []commonaddress.Address `json:"addresses"`
}

// V1DataAgentsIDPasswordPut is
// v1 data type request struct for
// /v1/agents/<agent-id>/password PUT
type V1DataAgentsIDPasswordPut struct {
	Password string `json:"password"`
}

// V1DataAgentsIDPermissionPut is
// v1 data type request struct for
// /v1/agents/<agent-id>/permission PUT
type V1DataAgentsIDPermissionPut struct {
	Permission uint64 `json:"permission"`
}

// V1DataAgentsIDTagIDsPut is
// v1 data type request struct for
// /v1/agents/<agent-id>/tag_ids PUT
type V1DataAgentsIDTagIDsPut struct {
	TagIDs []uuid.UUID `json:"tag_ids"`
}

// V1DataAgentsIDStatusPut is
// v1 data type request struct for
// /v1/agents/<agent-id>/status PUT
type V1DataAgentsIDStatusPut struct {
	Status string `json:"status"`
}

// V1DataAgentsIDDialPost is
// v1 data type request struct for
// /v1/agents/<agent-id>/dial PUT
type V1DataAgentsIDDialPost struct {
	Source       commonaddress.Address `json:"source"`
	FlowID       uuid.UUID             `json:"flow_id"`
	MasterCallID uuid.UUID             `json:"master_call_id"`
}
