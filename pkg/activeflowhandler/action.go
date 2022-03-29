package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

// appendActions append the action after the target action id.
func appendActions(af *activeflow.Activeflow, targetActionID uuid.UUID, act []action.Action) error {

	var res []action.Action

	// get idx
	idx := -1
	for i, act := range af.Actions {
		if act.ID == targetActionID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("could not find action index")
	}

	// append
	res = append(res, af.Actions[:idx+1]...)
	res = append(res, act...)
	res = append(res, af.Actions[idx+1:]...)

	af.Actions = res

	return nil
}

// replaceActions replaces the target action id to the given list of actions.
func replaceActions(af *activeflow.Activeflow, targetActionID uuid.UUID, act []action.Action) error {

	var res []action.Action

	// get idx
	idx := -1
	for i, act := range af.Actions {
		if act.ID == targetActionID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("could not find action index")
	}

	// append
	res = append(res, af.Actions[:idx]...)
	res = append(res, act...)
	res = append(res, af.Actions[idx+1:]...)

	af.Actions = res

	return nil
}

// getActionsFromFlow gets the actions from the flow.
func (h *activeflowHandler) getActionsFromFlow(ctx context.Context, flowID uuid.UUID, customerID uuid.UUID) ([]action.Action, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "getActionsFromFlow",
		},
	)

	// get flow
	f, err := h.reqHandler.FMV1FlowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get flow info. err: %v", err)
		return nil, err
	}

	if f.CustomerID != customerID {
		log.Errorf("The customer has no permission. customer_id: %d", customerID)
		return nil, fmt.Errorf("no flow found")
	}

	return f.Actions, nil
}

// getExitActionID returns exit action id
func (h *activeflowHandler) getExitActionID(actions []action.Action, actionID uuid.UUID) (uuid.UUID, error) {
	if len(actions) == 0 {
		return uuid.Nil, fmt.Errorf("empty actions")
	}

	var idx int
	for i, act := range actions {
		if act.ID == actionID {
			idx = i
			break
		}
	}

	if idx >= len(actions)-1 {
		return action.IDFinish, nil
	}
	return actions[idx+1].ID, nil
}

// removeAction removes action from the actions
//nolint:unused // this is ok
func (h *activeflowHandler) removeAction(actions []action.Action, actionID uuid.UUID) ([]action.Action, error) {
	for i, a := range actions {
		if a.ID == actionID {
			res := append(actions[:i], actions[i+1:]...)
			return res, nil
		}
	}

	return nil, fmt.Errorf("no action found")

}

// getNextAction returns next action from the active-flow
// It sets next action to current action.
func (h *activeflowHandler) getNextAction(ctx context.Context, id uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "getNextAction",
		"id":                id,
		"current_action_id": caID,
	})
	log.Debug("Getting next action.")

	// get active-flow
	af, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return nil, err
	}
	log = log.WithField("active_flow_current_action_id", af.CurrentAction.ID)
	log.WithField("active_flow", af).Debug("Found active flow.")

	// check execute count.
	if af.ExecuteCount > maxActiveFlowExecuteCount {
		log.Errorf("Exceed maximum action execution count. execute_count: %d", af.ExecuteCount)
		return nil, fmt.Errorf("exceed maximum action execution count")
	}

	// check the empty actions and action id is start id or not.
	switch {
	case len(af.Actions) == 0:
		log.Errorf("The action array is empty.")
		return nil, fmt.Errorf("action array is empty")

	case af.CurrentAction.ID == action.IDStart:
		resAction := af.Actions[0]
		return &resAction, nil
	}

	// compare current action.
	// if the current action does not match with the active-flow's current action,
	// discard it here
	if af.CurrentAction.ID != caID {
		log.Error("The current action does not match.")
		return nil, fmt.Errorf("current action does not match")
	}

	// check the fowrard action id.
	if af.ForwardActionID != action.IDEmpty {
		log.Debug("The forward action ID exist.")
		act := getActionByID(af.Actions, af.ForwardActionID)
		if act == nil {
			log.WithField("actions", af.Actions).Errorf("Could not find the forward action in the actions. forward_action_id: %v", af.ForwardActionID)
			return nil, fmt.Errorf("could not find forward action in the actions array")
		}
		return act, nil
	} else if af.CurrentAction.NextID != action.IDEmpty {
		log.Debug("The next action ID exist.")
		act := getActionByID(af.Actions, af.CurrentAction.NextID)
		if act == nil {
			log.WithField("actions", af.Actions).Errorf("Could not find the next action in the actions. next_action_id: %v", af.CurrentAction.NextID)
			return nil, fmt.Errorf("could not find move action in the actions array")
		}
		return act, nil
	}

	// get current action's index
	idx := 0
	found := false
	for _, act := range af.Actions {
		if act.ID == caID {
			found = true
			break
		}
		idx++
	}

	// get nextAction
	if !found || idx >= (len(af.Actions)-1) {
		// No more actions left.
		log.Infof("No more action left. found: %v, idx: %v", found, idx)
		return nil, fmt.Errorf("no more action left")
	}

	res := af.Actions[idx+1]
	return &res, nil
}

// getActionByID returns give id's action from the actions
func getActionByID(actions []action.Action, id uuid.UUID) *action.Action {
	for _, act := range actions {
		if act.ID == id {
			return &act
		}
	}
	return nil
}
