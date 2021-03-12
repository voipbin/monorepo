package fmflow

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmaction"
)

// Flow struct
type Flow struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Actions []fmaction.Action `json:"actions"`

	Persist    bool   `json:"persist"`
	WebhookURI string `json:"webhook_uri"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertFlow returns converted data from fmflow.Flow to flow.Flow
func (f *Flow) ConvertFlow() *models.Flow {

	actions := []models.Action{}
	for _, a := range f.Actions {
		actions = append(actions, *a.ConvertAction())
	}

	res := &models.Flow{
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
func CreateFlow(f *models.Flow) *Flow {

	actions := []fmaction.Action{}
	for _, a := range f.Actions {
		actions = append(actions, *fmaction.CreateAction(&a))
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
