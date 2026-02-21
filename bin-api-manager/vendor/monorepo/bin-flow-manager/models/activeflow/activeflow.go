package activeflow

import (
	"fmt"
	"reflect"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

// Activeflow struct
type Activeflow struct {
	commonidentity.Identity

	FlowID uuid.UUID `json:"flow_id,omitempty" db:"flow_id,uuid"`
	Status Status    `json:"status,omitempty" db:"status"`

	ReferenceType         ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID           uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`
	ReferenceActiveflowID uuid.UUID     `json:"reference_activeflow_id,omitempty" db:"reference_activeflow_id,uuid"` // the activeflow which created this activeflow by the on_complete_flow setting

	OnCompleteFlowID uuid.UUID `json:"on_complete_flow_id,omitempty" db:"on_complete_flow_id,uuid"` // will be triggered when this activeflow is completed or terminated

	// stack
	StackMap map[uuid.UUID]*stack.Stack `json:"stack_map,omitempty" db:"stack_map,json"`

	// current info
	CurrentStackID uuid.UUID     `json:"current_stack_id,omitempty" db:"current_stack_id,uuid"`
	CurrentAction  action.Action `json:"current_action,omitempty" db:"current_action,json"`

	// forward info
	ForwardStackID  uuid.UUID `json:"forward_stack_id,omitempty" db:"forward_stack_id,uuid"`
	ForwardActionID uuid.UUID `json:"forward_action_id,omitempty" db:"forward_action_id,uuid"`

	// execute
	ExecuteCount    uint64          `json:"execute_count,omitempty" db:"execute_count"`
	ExecutedActions []action.Action `json:"executed_actions,omitempty" db:"executed_actions,json"` // list of executed actions

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
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
	ReferenceTypeNone         ReferenceType = ""             // none
	ReferenceTypeAI           ReferenceType = "ai"           // ai
	ReferenceTypeAPI          ReferenceType = "api"          // api
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

var MapActionMediaTypeByReferenceType = map[ReferenceType]action.MediaType{
	ReferenceTypeNone: action.MediaTypeNone,

	ReferenceTypeCall: action.MediaTypeRealTimeCommunication,

	ReferenceTypeAI:           action.MediaTypeNonRealTimeCommunication,
	ReferenceTypeAPI:          action.MediaTypeNonRealTimeCommunication,
	ReferenceTypeConversation: action.MediaTypeNonRealTimeCommunication,
	ReferenceTypeRecording:    action.MediaTypeNonRealTimeCommunication,
	ReferenceTypeTranscribe:   action.MediaTypeNonRealTimeCommunication,
	ReferenceTypeCampaign:     action.MediaTypeNonRealTimeCommunication,
}
