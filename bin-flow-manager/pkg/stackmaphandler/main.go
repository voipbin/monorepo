package stackmaphandler

//go:generate mockgen -package stackmaphandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

const (
	maxStackCount = 100
)

// stackHandler defines
type stackHandler struct {
	utilHandler utilhandler.UtilHandler
}

// StackmapHandler defines
type StackmapHandler interface {
	Create(actions []action.Action) map[uuid.UUID]*stack.Stack

	AddActions(stackMap map[uuid.UUID]*stack.Stack, targetStackID uuid.UUID, targetActionID uuid.UUID, actions []action.Action) error
	GetAction(stackMap map[uuid.UUID]*stack.Stack, startStackID uuid.UUID, actionID uuid.UUID, releaseStack bool) (uuid.UUID, *action.Action, error)
	GetNextAction(stackMap map[uuid.UUID]*stack.Stack, currentStackID uuid.UUID, currentAction *action.Action, relaseStack bool) (uuid.UUID, *action.Action)

	PushStackByActions(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, actions []action.Action, currentStackID uuid.UUID, currentActionID uuid.UUID) (*stack.Stack, error)
	PopStack(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error)
}

// NewStackmapHandler returns a new StackHandler
func NewStackmapHandler() StackmapHandler {
	return &stackHandler{
		utilHandler: utilhandler.NewUtilHandler(),
	}
}
