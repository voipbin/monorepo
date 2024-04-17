package activeflow

import (
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

// Activeflow struct
type Activeflow struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	FlowID uuid.UUID `json:"flow_id"`
	Status Status    `json:"status"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	// stack
	StackMap map[uuid.UUID]*stack.Stack `json:"stack_map"`

	// current info
	CurrentStackID uuid.UUID     `json:"current_stack_id"`
	CurrentAction  action.Action `json:"current_action"`

	// forward info
	ForwardStackID  uuid.UUID `json:"forward_stack_id"`
	ForwardActionID uuid.UUID `json:"forward_action_id"`

	// execute
	ExecuteCount    uint64          `json:"execute_count"`
	ExecutedActions []action.Action `json:"executed_actions"` // list of executed actions

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Status define
type Status string

// list of Status
const (
	StatusNone    Status = ""
	StatusRunning Status = "running"
	StatusEnded   Status = "ended"
)

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
