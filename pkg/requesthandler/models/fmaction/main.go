package fmaction

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
)

// Action struct
type Action struct {
	ID     uuid.UUID       `json:"id"`
	Type   string          `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`
}

// ConvertAction return converted action.Action
func (r *Action) ConvertAction() *action.Action {
	return &action.Action{
		ID:     r.ID,
		Type:   action.Type(r.Type),
		Option: r.Option,
	}
}

// CreateAction returns created fmaction from the action.Action.
func CreateAction(a *action.Action) *Action {
	return &Action{
		ID:     a.ID,
		Type:   string(a.Type),
		Option: a.Option,
	}
}
