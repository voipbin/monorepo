package fmaction

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// Action struct
type Action struct {
	ID     uuid.UUID       `json:"id"`
	Type   string          `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`
}

// ConvertAction return converted action.Action
func (r *Action) ConvertAction() *models.Action {
	return &models.Action{
		ID:     r.ID,
		Type:   models.ActionType(r.Type),
		Option: r.Option,
	}
}

// CreateAction returns created fmaction from the action.Action.
func CreateAction(a *models.Action) *Action {
	return &Action{
		ID:     a.ID,
		Type:   string(a.Type),
		Option: a.Option,
	}
}
