package stackhandler

//go:generate mockgen -package stackhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

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

// StackHandler defines
type StackHandler interface {
	SearchAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, actionID uuid.UUID) (uuid.UUID, *action.Action, error)
	GetAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, currentStackID uuid.UUID, targetActionID uuid.UUID, releaseStack bool) (uuid.UUID, *action.Action, error)
	GetNextAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, currentStackID uuid.UUID, currentAction *action.Action, relaseStack bool) (uuid.UUID, *action.Action)

	Create(stackID uuid.UUID, actions []action.Action, returnStackID uuid.UUID, returnActionID uuid.UUID) *stack.Stack
	Get(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error)
	Push(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, actions []action.Action, currentStackID uuid.UUID, currentActionID uuid.UUID) (*stack.Stack, error)
	Pop(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error)
	InitStackMap(ctx context.Context, actions []action.Action) map[uuid.UUID]*stack.Stack

	StackMapInit(actions []action.Action) map[uuid.UUID]*stack.Stack
	StackMapGet(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error)
	StackMapPush(stackMap map[uuid.UUID]*stack.Stack, s *stack.Stack) error
	StackMapPop(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error)
	StackMapPushActions(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, targetActionID uuid.UUID, actions []action.Action) error
}

// NewStackHandler returns a new StackHandler
func NewStackHandler() StackHandler {
	return &stackHandler{
		utilHandler: utilhandler.NewUtilHandler(),
	}
}
