package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmaction"
)

// FMV1DataFlowPost is
// v1 data type request struct for
// /v1/flows POST
type FMV1DataFlowPost struct {
	ID     uuid.UUID `json:"id"`      // flow's id
	UserID uint64    `json:"user_id"` // flow's owner

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail

	Actions []action.Action `json:"actions"` // actions

	Persist bool `json:"persist"` // persist. If it is true, set the flow into the database.
}

// FMV1DataFlowIDPut is
// v1 data type request struct for
// /v1/flows/{id} PUT
type FMV1DataFlowIDPut struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail

	Actions []fmaction.Action `json:"actions"` // actions
}
