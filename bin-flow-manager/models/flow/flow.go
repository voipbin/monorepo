package flow

import (
	"fmt"
	"reflect"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-flow-manager/models/action"
)

// Flow struct
type Flow struct {
	commonidentity.Identity

	Type Type `json:"type,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Persist bool `json:"persist,omitempty"`

	Actions []action.Action `json:"actions,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldType Field = "type" // type

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldPersist Field = "persist" // persist

	FieldActions Field = "actions" // actions

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)

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
