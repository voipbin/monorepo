package activeflow

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
)

// ActiveFlow struct
type ActiveFlow struct {
	CallID        uuid.UUID
	FlowID        uuid.UUID
	UserID        uint64
	CurrentAction action.Action

	Actions []action.Action

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
