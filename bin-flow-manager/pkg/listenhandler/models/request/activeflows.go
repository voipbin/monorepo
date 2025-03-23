package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
)

// V1DataActiveFlowsPost is
// data type request struct for
// /v1/activeflows POST
type V1DataActiveFlowsPost struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	FlowID uuid.UUID `json:"flow_id"`

	ReferenceType         activeflow.ReferenceType `json:"reference_type"`
	ReferenceID           uuid.UUID                `json:"reference_id"`
	ReferenceActiveflowID uuid.UUID                `json:"reference_activeflow_id,omitempty"`
}

// V1DataActiveFlowsIDNextGet is
// data type request struct for
// /v1/activeflows/{id}/next GET
type V1DataActiveFlowsIDNextGet struct {
	CurrentActionID uuid.UUID `json:"current_action_id"`
}

// V1DataActiveFlowsIDForwardActionIDPut is
// data type request struct for
// /v1/activeflows/{id}/forward_action_id PUT
type V1DataActiveFlowsIDForwardActionIDPut struct {
	ForwardActionID uuid.UUID `json:"forward_action_id"`
	ForwardNow      bool      `json:"forward_now"`
}

// V1DataFlowIDActionsPut is
// data type request struct for
// /v1/flows/{id}/actions PUT
type V1DataFlowIDActionsPut struct {
	Actions []action.Action `json:"actions"` // actions
}

// V1DataActiveFlowsIDPushActionPost is
// data type request struct for
// /v1/activeflows/{id}/push_action POST
type V1DataActiveFlowsIDPushActionPost struct {
	Actions []action.Action `json:"actions"` // actions
}

// V1DataActiveFlowsIDAddActionPost is
// data type request struct for
// /v1/activeflows/{id}/add_action POST
type V1DataActiveFlowsIDAddActionPost struct {
	Actions []action.Action `json:"actions"` // actions
}

// V1DataActiveFlowsIDServiceStopPost is
// data type request struct for
// /v1/activeflows/{id}/service_stop POST
type V1DataActiveFlowsIDServiceStopPost struct {
	ServiceID uuid.UUID `json:"service_id"`
}
