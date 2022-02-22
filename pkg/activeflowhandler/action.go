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
func appendActions(af *activeflow.ActiveFlow, targetActionID uuid.UUID, act []action.Action) error {

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
func replaceActions(af *activeflow.ActiveFlow, targetActionID uuid.UUID, act []action.Action) error {

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
