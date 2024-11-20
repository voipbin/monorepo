package stackhandler

//go:generate mockgen -package stackhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

const (
	maxStackCount = 100
)

// stackHandler defines
type stackHandler struct{}

// StackHandler defines
type StackHandler interface {
	SearchAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, actionID uuid.UUID) (uuid.UUID, *action.Action, error)
	GetAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, currentStackID uuid.UUID, targetActionID uuid.UUID, releaseStack bool) (uuid.UUID, *action.Action, error)
	GetNextAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, currentStackID uuid.UUID, currentAction *action.Action, relaseStack bool) (uuid.UUID, *action.Action)

	Get(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error)
	Push(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, actions []action.Action, currentStackID uuid.UUID, currentActionID uuid.UUID) (uuid.UUID, *action.Action, error)
}

// NewStackHandler returns a new StackHandler
func NewStackHandler() StackHandler {
	return &stackHandler{}
}
