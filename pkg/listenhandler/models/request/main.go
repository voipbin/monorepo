package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

// V1DataActiveFlowsPost is
// v1 data type request struct for
// /v1/active-flows POST
type V1DataActiveFlowsPost struct {
	FlowID uuid.UUID `json:"flow_id"`

	ReferenceType activeflow.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID                `json:"reference_id"`
}

// V1DataActiveFlowsIDNextGet is
// v1 data type request struct for
// /v1/active-flows/{id}/next GET
type V1DataActiveFlowsIDNextGet struct {
	CurrentActionID uuid.UUID `json:"current_action_id"`
}

// V1DataActiveFlowsIDForwardActionIDPut is
// v1 data type request struct for
// /v1/active-flows/{id}/forward_action_id PUT
type V1DataActiveFlowsIDForwardActionIDPut struct {
	ForwardActionID uuid.UUID `json:"forward_action_id"`
	ForwardNow      bool      `json:"forward_now"`
}

// V1DataFlowPost is
// v1 data type request struct for
// /v1/flows POST
type V1DataFlowPost struct {
	CustomerID uuid.UUID `json:"customer_id"` // flow's owner
	Type       flow.Type `json:"type"`        // flow's type

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail

	Actions []action.Action `json:"actions"` // actions

	Persist bool `json:"persist"` // persist. If it is true, set the flow into the database.
}

// V1DataFlowIDPut is
// v1 data type request struct for
// /v1/flows/{id} PUT
type V1DataFlowIDPut struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail

	Actions []action.Action `json:"actions"` // actions
}
