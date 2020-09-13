package flow

import (
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/action"
)

// Flow struct
type Flow struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Persist bool `json:"persist"`

	Actions []action.Action `json:"actions"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Matches return true if the given items are the same
func (a *Flow) Matches(x interface{}) bool {
	comp := x.(*Flow)
	c := *a

	c.TMCreate = comp.TMCreate

	return reflect.DeepEqual(c, *comp)
}

func (a *Flow) String() string {
	return fmt.Sprintf("%v", *a)
}
