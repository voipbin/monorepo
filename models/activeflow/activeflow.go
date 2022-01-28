package activeflow

import (
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// ActiveFlow struct
type ActiveFlow struct {
	CallID     uuid.UUID `json:"call_id"`
	FlowID     uuid.UUID `json:"flow_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	WebhookURI string    `json:"webhook_uri"`

	CurrentAction   action.Action `json:"current_action"`
	ExecuteCount    uint64        `json:"execute_count"`
	ForwardActionID uuid.UUID     `json:"forward_action_id"`

	Actions []action.Action `json:"actions"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Matches return true if the given items are the same
// Used in test
func (a *ActiveFlow) Matches(x interface{}) bool {
	comp := x.(*ActiveFlow)
	c := *a

	c.TMCreate = comp.TMCreate
	c.TMUpdate = comp.TMUpdate
	c.TMDelete = comp.TMDelete

	return reflect.DeepEqual(c, *comp)
}

func (a *ActiveFlow) String() string {
	return fmt.Sprintf("%v", *a)
}
