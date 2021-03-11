package fmflow

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Flow struct
type Flow struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Actions []action.Action `json:"actions"`

	Persist bool `json:"persist"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
