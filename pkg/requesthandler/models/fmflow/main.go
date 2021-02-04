package fmflow

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmaction"
)

// Flow struct
type Flow struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Actions []fmaction.Action `json:"actions"`

	Persist bool `json:"persist"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertFlow returns converted data from fmflow.Flow to flow.Flow
func (f *Flow) ConvertFlow() *flow.Flow {

	res := &flow.Flow{
		ID:     f.ID,
		UserID: f.UserID,

		Name:   f.Name,
		Detail: f.Detail,

		Persist: f.Persist,

		Actions: []action.Action{},

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}

	// convert actions
	for _, a := range f.Actions {
		res.Actions = append(res.Actions, *a.ConvertAction())
	}

	return res
}
