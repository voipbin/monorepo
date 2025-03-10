package stackmaphandler

import (
	"fmt"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *stackHandler) CreateStack(stackID uuid.UUID, actions []action.Action, returnStackID uuid.UUID, returnActionID uuid.UUID) *stack.Stack {

	if stackID == uuid.Nil {
		stackID = h.utilHandler.UUIDCreate()
	}

	res := &stack.Stack{
		ID:             stackID,
		Actions:        actions,
		ReturnStackID:  returnStackID,
		ReturnActionID: returnActionID,
	}

	return res
}
func (h *stackHandler) DeleteStack(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) {

	if stackID == stack.IDMain {
		return
	}

	tmp, ok := stackMap[stackID]
	if !ok || tmp == nil {
		return
	}

	delete(stackMap, stackID)
}

func (h *stackHandler) GetStack(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error) {
	res, ok := stackMap[stackID]
	if !ok || res == nil {
		return nil, fmt.Errorf("no stack found. stack_id: %s", stackID)
	}

	return res, nil
}

// PushStackByActions creates a new stack and push it to the stackMap.
// it returns stack_id and action for next execution.
func (h *stackHandler) PushStackByActions(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, actions []action.Action, currentStackID uuid.UUID, currentActionID uuid.UUID) (*stack.Stack, error) {

	if stackID == uuid.Nil {
		stackID = h.utilHandler.UUIDCreate()
	}

	_, err := h.GetStack(stackMap, stackID)
	if err == nil {
		return nil, fmt.Errorf("stack already exists. stack_id: %s", stackID)
	}

	res := h.CreateStack(stackID, actions, currentStackID, currentActionID)

	stackMap[stackID] = res

	return res, nil
}

// PopStack pops the stack from the stackMap.
// it returns the return stack_id and action_id.
func (h *stackHandler) PopStack(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error) {
	if stackID == stack.IDMain {
		return nil, fmt.Errorf("cannot pop main stack")
	}

	res, err := h.GetStack(stackMap, stackID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the stack. stack_id: %s", stackID)
	}

	h.DeleteStack(stackMap, stackID)
	return res, nil
}
