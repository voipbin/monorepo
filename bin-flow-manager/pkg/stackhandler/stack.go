package stackhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
)

// FlowGet returns flow
func (h *stackHandler) create(ctx context.Context, stackID uuid.UUID, actions []action.Action, returnStackID uuid.UUID, returnActionID uuid.UUID) *stack.Stack {

	res := &stack.Stack{
		ID:             stackID,
		Actions:        actions,
		ReturnStackID:  returnStackID,
		ReturnActionID: returnActionID,
	}

	return res
}

func (h *stackHandler) remove(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID) {
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
func (h *stackHandler) Push(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, actions []action.Action, currentStackID uuid.UUID, currentActionID uuid.UUID) (uuid.UUID, *action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "Push",
		"current_stack_id":  currentStackID,
		"current_action_id": currentActionID,
	})
	log.WithField("action", actions).Debugf("Pushing a new stack.")

	// generate stackID
	var stackID uuid.UUID = uuid.Nil
	if len(stackMap) == 0 {
		stackID = stack.IDMain
	} else {
		exist := false
		for i := 0; i < 10; i++ {
			stackID = uuid.Must(uuid.NewV4())
			_, err := h.Get(ctx, stackMap, stackID)
			if err != nil {
				exist = false
				break
			}
			exist = true
		}
		if exist {
			return stack.IDEmpty, nil, fmt.Errorf("could not generate uniq stack_id")
		}
	}

	// create a new stack
	s := h.create(ctx, stackID, actions, currentStackID, currentActionID)
	log.WithField("stack", s).Debugf("Created a new stack. stack_id: %s", s.ID)

	// push it
	stackMap[stackID] = s

	return s.ID, &s.Actions[0], nil
}
