package stackhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

// actionFind returns a pointer of the given actionID's action from the actions
func (h *stackHandler) actionFind(actions []action.Action, actionID uuid.UUID) *action.Action {

	i := 0
	for _, a := range actions {
		if a.ID == actionID {
			// found
			return &actions[i]
		}
		i++
	}

	return nil
}

// SearchAction returns a pointer of the given action id's action.
// it checks all stacks from the given stackMap if the stackID is empty.
func (h *stackHandler) SearchAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, actionID uuid.UUID) (uuid.UUID, *action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "SearchAction",
		"action_id": actionID,
	})
	log.Debugf("Getting the action. action_id: %s", actionID)

	if stackID != stack.IDEmpty {
		// get stack
		s, err := h.Get(ctx, stackMap, stackID)

		if err != nil {
			log.Errorf("Could not find stack. err: %v", err)
			return stack.IDEmpty, nil, err
		}

		a := h.actionFind(s.Actions, actionID)
		if a == nil {
			return stack.IDEmpty, nil, fmt.Errorf("action not found")
		}

		return stackID, a, nil
	}

	// if stackID not specified, we run through all stacks
	for tmpStackID, s := range stackMap {

		a := h.actionFind(s.Actions, actionID)
		if a != nil {
			// found
			return tmpStackID, a, nil
		}
	}

	return stack.IDEmpty, nil, fmt.Errorf("action not found")
}

// GetAction returns given action id's action
// it follows stack's return addresses and release the memory when it gets out from the stack.
func (h *stackHandler) GetAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, currentStackID uuid.UUID, targetActionID uuid.UUID, releaseStack bool) (uuid.UUID, *action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "GetAction",
		"current_stack_id": currentStackID,
		"target_action_id": targetActionID,
	})
	log.Debugf("Getting the action. action_id: %s", targetActionID)

	resStackID := currentStackID
	for i := 0; i < maxStackCount; i++ {

		// get stack
		s, err := h.Get(ctx, stackMap, resStackID)
		if err != nil {
			log.Errorf("Could not find stack. err: %v", err)
			return stack.IDEmpty, nil, err
		}

		if currentStackID == stack.IDMain {
			if len(s.Actions) == 0 {
				return stack.IDEmpty, nil, fmt.Errorf("actions are empty")
			}

			if targetActionID == action.IDStart {
				return resStackID, &s.Actions[0], nil
			}
		}

		// get action
		tmpAction := h.actionFind(s.Actions, targetActionID)
		if tmpAction != nil {
			// found
			return resStackID, tmpAction, nil
		}

		resStackID = s.ReturnStackID
		if releaseStack {
			h.remove(stackMap, s.ID)
		}

		if resStackID == stack.IDEmpty {
			return stack.IDEmpty, nil, fmt.Errorf("no more stack left")
		}
	}

	return stack.IDEmpty, nil, fmt.Errorf("exceed mac stack count")
}

// GetNextAction returns next action.
// it checks all of related stacks.
// if it couldn't find next action, returns finish action.
func (h *stackHandler) GetNextAction(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, currentStackID uuid.UUID, currentAction *action.Action, relaseStack bool) (uuid.UUID, *action.Action) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "GetNextAction",
		"current_stack_id":  currentStackID,
		"current_action_id": currentAction.ID,
	})
	log.WithField("action", currentAction).Debugf("Getting next action.")

	// check the currrent stack_id is the main stack
	if currentStackID == stack.IDMain {
		s, err := h.Get(ctx, stackMap, currentStackID)
		if err != nil {
			log.Errorf("Could not find stack. err: %v", err)
			return stack.IDEmpty, &action.ActionFinish
		}

		if len(s.Actions) == 0 {
			log.Debugf("Actions are empty.")
			return stack.IDEmpty, &action.ActionFinish
		}

		if currentAction.ID == action.IDStart {
			return currentStackID, &s.Actions[0]
		}
	}

	// check next id.
	if currentAction.NextID != action.IDEmpty {
		resStackID, resAction, err := h.GetAction(ctx, stackMap, currentStackID, currentAction.NextID, true)
		if err != nil {
			log.Errorf("Could not get action. err: %v", err)
			return stack.IDEmpty, &action.ActionFinish
		}

		return resStackID, resAction
	}

	// var tmpAction *action.Action = nil
	tmpStackID := currentStackID
	tmpActionID := currentAction.ID
	for i := 0; i < maxStackCount; i++ {

		// get stack
		s, err := h.Get(ctx, stackMap, tmpStackID)
		if err != nil {
			log.Errorf("Could not find stack. err: %v", err)
			return stack.IDEmpty, &action.ActionFinish
		}

		// get action
		found := false
		idx := 0
		for j, a := range s.Actions {
			if a.ID == tmpActionID {
				found = true
				idx = j
				break
			}
		}

		if !found {
			log.Errorf("Could not find action in the stack.")
			return stack.IDEmpty, &action.ActionFinish
		}

		if idx < (len(s.Actions) - 1) {
			tmpAction := s.Actions[idx]
			if tmpAction.NextID != action.IDEmpty {

				resStackID, resAction, err := h.GetAction(ctx, stackMap, tmpStackID, tmpAction.NextID, true)
				if err != nil {
					log.Errorf("Could not get action for next_id. err: %v", err)
					return stack.IDEmpty, &action.ActionFinish
				}

				return resStackID, resAction
			}

			resAction := &s.Actions[idx+1]
			return tmpStackID, resAction
		}

		// the found action is placed in the end of actions.
		// we have to take a look the return stack
		tmpStackID = s.ReturnStackID
		tmpActionID = s.ReturnActionID

		if relaseStack {
			h.remove(stackMap, s.ID)
		}

		if tmpStackID == stack.IDEmpty {
			// no more return stack left
			log.Debugf("No more return stack left.")
			return stack.IDMain, &action.ActionFinish
		}
	}

	log.Errorf("Exceed max stack count.")
	return stack.IDMain, &action.ActionFinish
}

// PushActions
func (h *stackHandler) PushActions(ctx context.Context, stackMap map[uuid.UUID]*stack.Stack, stackID uuid.UUID, targetActionID uuid.UUID, actions []action.Action) (map[uuid.UUID]*stack.Stack, error) {

	tmp, err := h.Get(ctx, stackMap, stackID)
	if err != nil {
		return nil, fmt.Errorf("no stack found. stack_id: %s", stackID)
	}

	for i, a := range tmp.Actions {
		if a.ID == targetActionID {
			tmp.Actions = append(tmp.Actions[:i+1], append(actions, tmp.Actions[i+1:]...)...)
			break
		}
	}

	return stackMap, nil
}
