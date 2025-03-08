package stackhandler

import (
	"fmt"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// StackMapInit initializes the stackmap with the main stack.
func (h *stackHandler) StackMapInit(actions []action.Action) map[uuid.UUID]*stack.Stack {

	stackmap := make(map[uuid.UUID]*stack.Stack)

	stackmap[stack.IDMain] = h.Create(stack.IDMain, actions, stack.IDEmpty, action.IDEmpty)

	return stackmap
}

// StackMapGet returns the stack from the stackmap.
func (h *stackHandler) StackMapGet(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error) {

	res, ok := stackMap[stackID]
	if !ok || res == nil {
		return nil, fmt.Errorf("no stack found. stack_id: %s", stackID)
	}

	return res, nil
}

// StackMapPush pushes the stack to the stackmap.
func (h *stackHandler) StackMapPush(stackMap map[uuid.UUID]*stack.Stack, s *stack.Stack) error {

	tmp, err := h.StackMapGet(stackMap, s.ID)
	if err == nil || tmp != nil {
		return errors.New("stack already exists")
	}

	stackMap[s.ID] = s

	return nil
}

// stackMapRemove pushes the stack to the stackmap.
func (h *stackHandler) stackMapRemove(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) {
	tmp, ok := stackMap[stackID]
	if !ok || tmp == nil {
		return
	}

	delete(stackMap, stackID)
}

// StackMapPop pops the stack from the stackmap.
// returns the popped stack.
func (h *stackHandler) StackMapPop(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) (*stack.Stack, error) {
	if stackID == stack.IDMain {
		return nil, fmt.Errorf("cannot pop main stack")
	}

	res, err := h.StackMapGet(stackMap, stackID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the stack. stack_id: %s", stackID)
	}

	h.stackMapRemove(stackMap, stackID)
	return res, nil
}

// StackMapPushActions pushes the actions to the given stack id after target action id.
func (h *stackHandler) StackMapPushActions(stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, targetActionID uuid.UUID, actions []action.Action) error {
	tmp, err := h.StackMapGet(stackMap, stackID)
	if err != nil {
		return fmt.Errorf("no stack found. stack_id: %s", stackID)
	}

	for i, a := range tmp.Actions {
		if a.ID == targetActionID {
			tmp.Actions = append(tmp.Actions[:i+1], append(actions, tmp.Actions[i+1:]...)...)
			break
		}
	}

	return nil
}
