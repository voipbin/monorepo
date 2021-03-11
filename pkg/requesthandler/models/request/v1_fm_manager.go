package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
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
