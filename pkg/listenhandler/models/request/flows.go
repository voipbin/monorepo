package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

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
