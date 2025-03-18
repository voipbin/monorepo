package stackmaphandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

// findAction returns a pointer of the given actionID's action from the actions
func (h *stackHandler) findAction(actions []action.Action, actionID uuid.UUID) (int, *action.Action) {

	i := 0
	for _, a := range actions {
		if a.ID == actionID {
			// found
			return i, &actions[i]
		}
		i++
	}

	return -1, nil
}

// GetAction returns given action id's action
// it follows stack's return addresses and release the memory when it gets out from the stack.
// it retuns stack id and action.
func (h *stackHandler) GetAction(stackMap map[uuid.UUID]*stack.Stack, startStackID uuid.UUID, actionID uuid.UUID, releaseStack bool) (uuid.UUID, *action.Action, error) {
	if startStackID == stack.IDEmpty {
		return stack.IDEmpty, nil, fmt.Errorf("invalid stack id")
	}
	if actionID == action.IDEmpty {
		return stack.IDEmpty, nil, fmt.Errorf("invalid action id")
	}

	tmpStackID := startStackID
	for range maxStackCount {

		if tmpStackID == stack.IDEmpty {
			return stack.IDEmpty, nil, fmt.Errorf("no more stack left")
		}

		// get stack
		s, err := h.GetStack(stackMap, tmpStackID)
		if err != nil {
			return stack.IDEmpty, nil, errors.Wrapf(err, "could not find stack. stack_id: %s", tmpStackID)
		}

		if startStackID == stack.IDMain {
			if len(s.Actions) == 0 {
				return stack.IDEmpty, nil, fmt.Errorf("actions are empty")
			}

			if actionID == action.IDStart {
				return tmpStackID, &s.Actions[0], nil
			}
		}

		// get action
		_, tmpAction := h.findAction(s.Actions, actionID)
		if tmpAction != nil {
			// found
			return tmpStackID, tmpAction, nil
		}

		tmpStackID = s.ReturnStackID
		if releaseStack {
			h.DeleteStack(stackMap, s.ID)
		}
	}

	return stack.IDEmpty, nil, fmt.Errorf("exceed mac stack count")
}

// GetNextAction returns next action.
// it checks all of related stacks.
// if it couldn't find next action, returns finish action.
func (h *stackHandler) GetNextAction(stackMap map[uuid.UUID]*stack.Stack, currentStackID uuid.UUID, currentAction *action.Action, relaseStack bool) (uuid.UUID, *action.Action) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "GetNextAction",
		"current_stack_id":  currentStackID,
		"current_action_id": currentAction.ID,
	})
	log.WithField("action", currentAction).Debugf("Getting next action.")

	tmpStackID := currentStackID
	tmpActionID := currentAction.ID
	for range maxStackCount {

		if tmpStackID == stack.IDEmpty {
			// no more return stack left
			return stack.IDMain, &action.ActionFinish
		}

		// get stack
		s, err := h.GetStack(stackMap, tmpStackID)
		if err != nil {
			//stack not found
			log.Infof("Could not find stack. err: %v", err)
			return stack.IDEmpty, &action.ActionFinish
		}

		if tmpActionID == action.IDStart {
			// start action
			return tmpStackID, &s.Actions[0]
		}

		idx, a := h.findAction(s.Actions, tmpActionID)
		if a == nil {
			//action not found
			log.Infof("Could not find action in the stack.")
			return stack.IDEmpty, &action.ActionFinish
		}

		// next id is not empty. get the next action
		if a.NextID != action.IDEmpty {
			resStackID, resAction, err := h.GetAction(stackMap, tmpStackID, a.NextID, true)
			if err != nil {
				//action not found
				log.Infof("Could not get action for next_id. next_id: %s, err: %v", a.NextID, err)
				return stack.IDEmpty, &action.ActionFinish
			}

			return resStackID, resAction
		}

		// check the action is the last action in the stack
		if idx < (len(s.Actions) - 1) {
			tmpAction := s.Actions[idx+1]
			resStackID, resAction, err := h.GetAction(stackMap, tmpStackID, tmpAction.ID, true)
			if err != nil {
				//action not found
				return stack.IDEmpty, &action.ActionFinish
			}

			return resStackID, resAction
		}

		// the found action is placed in the end of actions.
		// we have to take a look the return stack
		tmpStackID = s.ReturnStackID
		tmpActionID = s.ReturnActionID

		if relaseStack {
			h.DeleteStack(stackMap, s.ID)
		}
	}

	// exceed max stack count
	return stack.IDMain, &action.ActionFinish
}

// AddActions adds actions to the stack.
func (h *stackHandler) AddActions(stackMap map[uuid.UUID]*stack.Stack, targetStackID uuid.UUID, targetActionID uuid.UUID, actions []action.Action) error {
	if targetStackID == stack.IDEmpty {
		return fmt.Errorf("invalid stack id")
	}
	if targetActionID == action.IDEmpty {
		return fmt.Errorf("invalid target action id")
	}

	s, err := h.GetStack(stackMap, targetStackID)
	if err != nil {
		return errors.Wrapf(err, "could not get stack. stack_id: %s", targetStackID)
	}

	idx, act := h.findAction(s.Actions, targetActionID)
	if act == nil {
		return fmt.Errorf("could not find action. action_id: %s", targetActionID)
	}

	// inject actions
	tmp := append(s.Actions[:idx+1], actions...)
	s.Actions = append(tmp, s.Actions[idx+1:]...)

	return nil
}
