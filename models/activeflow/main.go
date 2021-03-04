package activeflow

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/action"
)

// ActiveFlow struct
type ActiveFlow struct {
	CallID        uuid.UUID     `json:"call_id"`
	FlowID        uuid.UUID     `json:"flow_id"`
	UserID        uint64        `json:"user_id"`
	CurrentAction action.Action `json:"current_action"`

	Actions []action.Action `json:"actions"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
