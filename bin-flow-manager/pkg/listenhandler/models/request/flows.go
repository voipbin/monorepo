package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
)

// V1DataFlowsPost is
// v1 data type request struct for
// /v1/flows POST
type V1DataFlowsPost struct {
	CustomerID uuid.UUID `json:"customer_id"` // flow's owner
	Type       flow.Type `json:"type"`        // flow's type

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail

	Actions []action.Action `json:"actions"` // actions

	Persist bool `json:"persist"` // persist. If it is true, set the flow into the database.
}

// V1DataFlowsIDPut is
// v1 data type request struct for
// /v1/flows/{id} PUT
type V1DataFlowsIDPut struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail

	Actions []action.Action `json:"actions"` // actions
}
