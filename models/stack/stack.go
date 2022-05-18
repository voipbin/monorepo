package stack

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Stack defines
type Stack struct {
	ID      uuid.UUID       `json:"id"`
	Actions []action.Action `json:"actions"`

	ReturnStackID  uuid.UUID `json:"return_stack_id"`
	ReturnActionID uuid.UUID `json:"return_action_id"`
}

// list of predefined stack ids.
var (
	IDEmpty uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000") //
	IDMain  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001") // main stack
)
