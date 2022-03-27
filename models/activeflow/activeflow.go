package activeflow

import (
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Activeflow struct
type Activeflow struct {
	ID uuid.UUID `json:"id"`

	CustomerID uuid.UUID `json:"customer_id"`
	FlowID     uuid.UUID `json:"flow_id"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	CurrentAction   action.Action `json:"current_action"`
	ExecuteCount    uint64        `json:"execute_count"`
	ForwardActionID uuid.UUID     `json:"forward_action_id"`

	Actions         []action.Action `json:"actions"`
	ExecutedActions []action.Action `json:"executed_actions"` // list of executed actions

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of ReferenceType
const (
	ReferenceTypeNone    ReferenceType = ""        // none
	ReferenceTypeCall    ReferenceType = "call"    // call
	ReferenceTypeMessage ReferenceType = "message" // message
)

// Matches return true if the given items are the same
// Used in test
func (a *Activeflow) Matches(x interface{}) bool {
	comp := x.(*Activeflow)
	c := *a

	c.TMCreate = comp.TMCreate
	c.TMUpdate = comp.TMUpdate
	c.TMDelete = comp.TMDelete

	return reflect.DeepEqual(c, *comp)
}

func (a *Activeflow) String() string {
	return fmt.Sprintf("%v", *a)
}
