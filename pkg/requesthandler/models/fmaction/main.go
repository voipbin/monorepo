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
	Next   uuid.UUID       `json:"next"`
}

// ConvertAction return converted action.Action
func (r *Action) ConvertAction() *action.Action {

	return &action.Action{
		ID:     r.ID,
		Type:   action.Type(r.Type),
		Option: r.Option,
	}
}
