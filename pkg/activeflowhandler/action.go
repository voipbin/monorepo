package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
)

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

// getNextAction returns next action from the active-flow
// It sets next action to current action.
func (h *activeflowHandler) getNextAction(ctx context.Context, activeflowID uuid.UUID, caID uuid.UUID) (uuid.UUID, *action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "getNextAction",
		"activeflow_id":     activeflowID,
		"current_action_id": caID,
	})
	log.Debug("Getting next action.")

	// get active-flow
	af, err := h.Get(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return stack.IDEmpty, nil, err
	}
	log = log.WithField("active_flow_current_action_id", af.CurrentAction.ID)
	log.WithField("active_flow", af).Debug("Found active flow.")

	// check execute count.
	if af.ExecuteCount > maxActiveFlowExecuteCount {
		log.Errorf("Exceed maximum action execution count. execute_count: %d", af.ExecuteCount)
		return stack.IDEmpty, nil, fmt.Errorf("exceed maximum action execution count")
	}

	if af.CurrentAction.ID != action.IDEmpty && af.CurrentAction.ID != caID {
		log.Error("The current action does not match.")
		return stack.IDEmpty, nil, fmt.Errorf("current action does not match")
	}

	// get next action
	var stackID uuid.UUID
	var act *action.Action
	if af.ForwardStackID != stack.IDEmpty && af.ForwardActionID != action.IDEmpty {
		log.Debugf("The forward action ID exist. forward_stack_id: %s, forward_action_id: %s", af.ForwardStackID, af.ForwardActionID)
		stackID, act, err = h.stackHandler.GetAction(ctx, af.StackMap, af.ForwardStackID, af.ForwardActionID, true)
		if err != nil {
			log.Errorf("Could not get action. err: %v", err)
			return stack.IDEmpty, nil, err
		}
	} else {
		log.Debugf("The forward action ID does not exist. current_stack_id: %s, current_action_id: %s", af.CurrentStackID, &af.CurrentAction.ID)
		stackID, act = h.stackHandler.GetNextAction(ctx, af.StackMap, af.CurrentStackID, &af.CurrentAction, true)
	}
	log.Debugf("Found next action. stack_id: %s, action_id: %s", stackID, act.ID)

	return stackID, act, nil
}
