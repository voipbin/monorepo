package activeflow

import (
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

// Activeflow struct
type Activeflow struct {
	commonidentity.Identity

	FlowID uuid.UUID `json:"flow_id,omitempty"`
	Status Status    `json:"status,omitempty"`

	ReferenceType         ReferenceType `json:"reference_type,omitempty"`
	ReferenceID           uuid.UUID     `json:"reference_id,omitempty"`
	ReferenceActiveflowID uuid.UUID     `json:"reference_activeflow_id,omitempty"`

	// stack
	StackMap map[uuid.UUID]*stack.Stack `json:"stack_map,omitempty"`

	// current info
	CurrentStackID uuid.UUID     `json:"current_stack_id,omitempty"`
	CurrentAction  action.Action `json:"current_action,omitempty"`

	// forward info
	ForwardStackID  uuid.UUID `json:"forward_stack_id,omitempty"`
	ForwardActionID uuid.UUID `json:"forward_action_id,omitempty"`

	// execute
	ExecuteCount    uint64          `json:"execute_count,omitempty"`
	ExecutedActions []action.Action `json:"executed_actions,omitempty"` // list of executed actions

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldFlowID Field = "flow_id" // flow_id
	FieldStatus Field = "status"  // status

	FieldReferenceType         Field = "reference_type"          // reference_type
	FieldReferenceID           Field = "reference_id"            // reference_id
	FieldReferenceActiveflowID Field = "reference_activeflow_id" // reference_activeflow_id

	FieldStackMap       Field = "stack_map"        // stack_map
	FieldCurrentStackID Field = "current_stack_id" // current_stack_id
	FieldCurrentAction  Field = "current_action"   // current_action

	FieldForwardStackID  Field = "forward_stack_id"  // forward_stack_id
	FieldForwardActionID Field = "forward_action_id" // forward_action_id

	FieldExecuteCount    Field = "execute_count"    // execute_count
	FieldExecutedActions Field = "executed_actions" // executed_actions

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)

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
	ReferenceTypeNone         ReferenceType = ""             // none
	ReferenceTypeAI           ReferenceType = "ai"           // ai
	ReferenceTypeCall         ReferenceType = "call"         // call
	ReferenceTypeCampaign     ReferenceType = "campaign"     // campaign
	ReferenceTypeConversation ReferenceType = "conversation" // conversation
	ReferenceTypeTranscribe   ReferenceType = "transcribe"   // transcribe
	ReferenceTypeRecording    ReferenceType = "recording"    // recording
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
