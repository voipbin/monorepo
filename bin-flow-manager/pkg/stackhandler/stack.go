package stackhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

// FlowGet returns flow
func (h *stackHandler) create(stackID uuid.UUID, actions []action.Action, returnStackID uuid.UUID, returnActionID uuid.UUID) *stack.Stack {

	res := &stack.Stack{
		ID:             stackID,
		Actions:        actions,
		ReturnStackID:  returnStackID,
		ReturnActionID: returnActionID,
	}

	return res
}

func (h *stackHandler) remove(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) {
	tmp, ok := stackMap[stackID]
	if !ok || tmp == nil {
		return
	}

	delete(stackMap, stackID)
}

func (h *stackHandler) Get(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error) {
	res, ok := stackMap[stackID]
	if !ok || res == nil {
		return nil, fmt.Errorf("no stack found. stack_id: %s", stackID)
	}

	return res, nil
}

// Push creates a new stack and push it to the stackMap.
// it returns stack_id and action for next execution.
func (h *stackHandler) Push(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, actions []action.Action, currentStackID uuid.UUID, currentActionID uuid.UUID) (uuid.UUID, *action.Action, error) {

	if stackID == uuid.Nil {
		stackID = h.utilHandler.UUIDCreate()
	}

	_, err := h.Get(ctx, stackMap, stackID)
	if err == nil {
		return stack.IDEmpty, nil, fmt.Errorf("stack already exists. stack_id: %s", stackID)
	}

	tmp := h.create(stackID, actions, currentStackID, currentActionID)

	stackMap[stackID] = tmp

	return tmp.ID, &tmp.Actions[0], nil
}

func (h *stackHandler) InitStackMap(ctx context.Context, actions []action.Action) map[uuid.UUID]*stack.Stack {
	tmp := h.create(stack.IDMain, actions, stack.IDEmpty, action.IDEmpty)
	res := map[uuid.UUID]*stack.Stack{
		stack.IDMain: tmp,
	}

	return res
}
