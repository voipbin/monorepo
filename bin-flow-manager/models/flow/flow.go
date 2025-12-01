package flow

import (
	"fmt"
	"reflect"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// Flow struct
type Flow struct {
	commonidentity.Identity

	Type Type `json:"type,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Persist bool `json:"persist,omitempty"`

	Actions []action.Action `json:"actions,omitempty"`

	OnCompleteFlowID uuid.UUID `json:"on_complete_flow_id,omitempty"` // will be triggered when this flow is completed

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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
