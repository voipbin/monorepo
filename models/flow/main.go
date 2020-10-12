package flow

import (
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
)

// Flow struct for client show
type Flow struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"-"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Actions []action.Action `json:"actions"`

	Persist bool `json:"-"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
