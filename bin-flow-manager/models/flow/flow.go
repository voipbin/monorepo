package flow

import (
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Flow struct
type Flow struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Type       Type      `json:"type"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Persist bool `json:"persist"`

	Actions []action.Action `json:"actions"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type defines
type Type string

// list of types
const (
	TypeNone       Type = ""
	TypeFlow       Type = "flow"       // normal flow
	TypeConference Type = "conference" // conference-manager
	TypeQueue      Type = "queue"      // queue-manager
	TypeCampaign   Type = "campaign"   // campaign-manager
	TypeTransfer   Type = "transfer"   // transfer-manager
)

// Matches return true if the given items are the same
func (a *Flow) Matches(x interface{}) bool {
	comp := x.(*Flow)
	c := *a

	c.TMCreate = comp.TMCreate
	c.TMUpdate = comp.TMUpdate

	return reflect.DeepEqual(c, *comp)
}

func (a *Flow) String() string {
	return fmt.Sprintf("%v", *a)
}
