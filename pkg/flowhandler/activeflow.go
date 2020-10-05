package flowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/activeflow"
)

// FlowCreate creates a flow
// func (h *flowHandler) ActiveFlowCreate(ctx context.Context, flow *flow.Flow, persist bool) (*flow.Flow, error) {
func (h *flowHandler) ActiveFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error) {

	// get flow
	flow, err := h.db.FlowGet(ctx, flowID)
	if err != nil {
		logrus.Errorf("Could not get the flow. err: %v", err)
		return nil, err
	}

	// create activeflow
	curTime := getCurTime()
	tmpAF := &activeflow.ActiveFlow{
		CallID: callID,
		FlowID: flowID,

		CurrentAction: action.Action{
			ID: action.IDStart,
		},

		Actions: flow.Actions,

		TMCreate: curTime,
		TMUpdate: curTime,
	}
	if err := h.db.ActiveFlowCreate(ctx, tmpAF); err != nil {
		return nil, err
	}

	// get created active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		return nil, err
	}

	return af, nil
}

// ActiveFlowNextActionGet returns next action from the active-flow
// It sets next action to current action.
func (h *flowHandler) ActiveFlowNextActionGet(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"call":              callID,
		"current_action_id": caID,
	})

	// get active-flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return nil, err
	}

	// compare current action.
	// if the current action does not match with the active-flow's current action,
	// discard it here
	if af.CurrentAction.ID != caID {
		log.Error("The current action does not match.")
		return nil, fmt.Errorf("current action does not match")
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
	var nextAction action.Action
	if found == false || idx >= (len(af.Actions)-1) {
		// check if the no more actions left, return finishID here.
		log.Infof("No more action left. found: %v, idx: %v", found, idx)

		// create finish hangup
		nextAction = action.Action{
			ID: action.IDFinish,
		}
	} else {
		nextAction = af.Actions[idx+1]
	}

	// update current action in active-flow
	if err := h.activeFlowUpdateCurrentAction(ctx, callID, &nextAction); err != nil {
		log.Errorf("Could not update the current action. err: %v", err)
		return nil, fmt.Errorf("could not update the current action. err: %v", err)
	}

	return &nextAction, nil
}

// activeFlowUpdateCurrentAction updates the current action in active-flow.
func (h *flowHandler) activeFlowUpdateCurrentAction(ctx context.Context, callID uuid.UUID, action *action.Action) error {
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": callID,
			"action":  action,
		},
	)

	// get af
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return err
	}

	// set current Action
	af.CurrentAction = *action
	af.TMUpdate = getCurTime()

	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active-flow's current action. err: %v", err)
		return err
	}

	return nil
}
