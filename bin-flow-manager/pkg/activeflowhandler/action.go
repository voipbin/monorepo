package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
)

// getActionsFromFlow gets the actions from the flow.
func (h *activeflowHandler) getActionsFromFlow(ctx context.Context, flowID uuid.UUID, customerID uuid.UUID) ([]action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getActionsFromFlow",
		"flow_id":     flowID,
		"customer_id": customerID,
	})

	// get flow
	f, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
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

// updateNextAction updates the next action to the current action.
// It sets next action to current action.
func (h *activeflowHandler) updateNextAction(ctx context.Context, activeflowID uuid.UUID, caID uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                      "updateNextAction",
		"activeflow_id":             activeflowID,
		"request_current_action_id": caID,
	})
	log.Debug("Getting next action.")

	// get activeflow with lock
	af, err := h.GetWithLock(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}
	defer func() {
		_ = h.ReleaseLock(ctx, activeflowID)
	}()

	// check execute count.
	if af.ExecuteCount > maxActiveFlowExecuteCount {
		log.Errorf("Exceeded maximum action execution count. execute_count: %d", af.ExecuteCount)
		return nil, fmt.Errorf("exceed maximum action execution count")
	}

	if af.Status == activeflow.StatusEnded {
		log.Debugf("The activeflow ended.")
		return nil, fmt.Errorf("the activeflow ended")
	}

	if af.CurrentAction.ID != action.IDEmpty && af.CurrentAction.ID != caID {
		log.WithField("activeflow.current_action_id", af.CurrentAction.ID).Error("The current action info does not match.")
		return nil, fmt.Errorf("current action does not match")
	}

	// get next action
	var resStackID uuid.UUID
	var resAct *action.Action
	if af.ForwardStackID != stack.IDEmpty && af.ForwardActionID != action.IDEmpty {
		log.Debugf("The forward action ID exist. forward_stack_id: %s, forward_action_id: %s", af.ForwardStackID, af.ForwardActionID)
		resStackID, resAct, err = h.stackHandler.GetAction(ctx, af.StackMap, af.ForwardStackID, af.ForwardActionID, true)
		if err != nil {
			log.Errorf("Could not get action. err: %v", err)
			return nil, err
		}
	} else {
		log.Debugf("The forward action ID does not exist. current_stack_id: %s, current_action_id: %s", af.CurrentStackID, &af.CurrentAction.ID)
		resStackID, resAct = h.stackHandler.GetNextAction(ctx, af.StackMap, af.CurrentStackID, &af.CurrentAction, true)
	}
	log.Debugf("Found next action. stack_id: %s, action_id: %s, action_type: %s", resStackID, resAct.ID, resAct.Type)

	// substitute the option variables.
	v, err := h.variableHandler.Get(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not get variables. err: %v", err)
		return nil, err
	}
	resAct.Option = h.variableHandler.SubstituteByte(ctx, resAct.Option, v)

	// update current action in activeflow
	res, err := h.updateCurrentAction(ctx, activeflowID, resStackID, resAct)
	if err != nil {
		log.Errorf("Could not update the current action. err: %v", err)
		return nil, fmt.Errorf("could not update the current action. err: %v", err)
	}
	log.WithField("action", res.CurrentAction).Debugf("Updated current action. action_type: %s", res.CurrentAction.Type)

	return res, nil
}
