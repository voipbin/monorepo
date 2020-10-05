package activeflow

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
)

// ActiveFlow struct
type ActiveFlow struct {
	CallID        uuid.UUID
	FlowID        uuid.UUID
	CurrentAction action.Action

	Actions []action.Action

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
