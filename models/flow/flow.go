package flow

import (
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

// Flow struct for client show
type Flow struct {
	ID     uuid.UUID `json:"id"` // Flow's ID
	UserID uint64    `json:"-"`  // Flow owner's User ID

	Name   string `json:"name"`   // Name
	Detail string `json:"detail"` // Detail

	Actions []action.Action `json:"actions"` // Actions

	Persist    bool   `json:"-"` // Persist
	WebhookURI string `json:"webhook_uri"`

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// ConvertFlow returns converted data from fmflow.Flow to flow.Flow
func ConvertFlow(f *fmflow.Flow) *Flow {

	actions := []action.Action{}
	for _, a := range f.Actions {
		tmp := action.ConvertAction(&a)

		actions = append(actions, *tmp)
	}

	res := &Flow{
		ID:     f.ID,
		UserID: f.UserID,

		Name:   f.Name,
		Detail: f.Detail,

		Persist:    f.Persist,
		WebhookURI: f.WebhookURI,

		Actions: actions,

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}

	return res
}

// CreateFlow returns converted data from flow.Flow to fmflow.Flow
func CreateFlow(f *Flow) *fmflow.Flow {

	actions := []fmaction.Action{}
	for _, a := range f.Actions {
		actions = append(actions, *action.CreateAction(&a))
	}

	res := &fmflow.Flow{
		ID:     f.ID,
		UserID: f.UserID,

		Name:   f.Name,
		Detail: f.Detail,

		Persist:    f.Persist,
		WebhookURI: f.WebhookURI,

		Actions: actions,

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}

	return res
}
