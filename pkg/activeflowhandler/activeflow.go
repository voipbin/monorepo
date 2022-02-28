package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

// ActiveFlowCreate creates a active flow
func (h *activeflowHandler) ActiveFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ActiveFlowCreate",
		"call_id": callID,
		"flow_id": flowID,
	})

	// get flow
	f, err := h.db.FlowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get the flow. err: %v", err)
		return nil, err
	}

	// create activeflow
	curTime := dbhandler.GetCurTime()
	tmpAF := &activeflow.ActiveFlow{
		CallID:     callID,
		FlowID:     flowID,
		CustomerID: f.CustomerID,

		CurrentAction: action.Action{
			ID: action.IDStart,
		},
		ExecuteCount:    0,
		ForwardActionID: action.IDEmpty,

		Actions:         f.Actions,
		ExecutedActions: []action.Action{},

		TMCreate: curTime,
		TMUpdate: curTime,
	}
	if err := h.db.ActiveFlowCreate(ctx, tmpAF); err != nil {
		log.Errorf("Could not create the active flow. err: %v", err)
		return nil, err
	}

	// get created active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get created active flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveFlowCreated, af)

	return af, nil
}

// ActiveFlowSetForwardActionID sets the forward action id of the call.
func (h *activeflowHandler) ActiveFlowSetForwardActionID(ctx context.Context, callID uuid.UUID, actionID uuid.UUID, forwardNow bool) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ActiveFlowSetForwardActionID",
		"call_id":           callID,
		"forward_action_id": actionID,
		"forward_now":       forwardNow,
	})

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return err
	}
	log.WithField("active_flow", af).Debug("Found active flow.")

	// check the action ID exists in the actions.
	found := false
	for _, a := range af.Actions {
		if a.ID == actionID {
			found = true
			break
		}
	}
	if !found {
		log.Errorf("Could not find foward action id in the actions.")
		return fmt.Errorf("foward action id not found")
	}

	// update active flow
	af.ForwardActionID = actionID
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow.")
		return err
	}

	// send action next
	if forwardNow {
		if err := h.reqHandler.CMV1CallActionNext(ctx, callID, true); err != nil {
			log.Errorf("Could not send action next request. err: %v", err)
			return err
		}
	}

	return nil
}

// ActiveFlowNextActionGet returns next action from the active-flow
// It sets next action to current action.
func (h *activeflowHandler) ActiveFlowNextActionGet(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ActiveFlowNextActionGet",
		"call_id":           callID,
		"current_action_id": caID,
	})

	// get next action from the active
	nextAction, err := h.getNextAction(ctx, callID, caID)
	if err != nil {
		log.Errorf("Could not get next action. err: %v", err)
		return nil, err
	}
	log.WithField("action", nextAction).Debug("Found next action.")

	// execute the active action
	res, err := h.executeAction(ctx, callID, nextAction)
	if err != nil {
		log.Errorf("Could not execute the active action. err: %v", err)
		return nil, err
	}

	return res, nil
}

// executeAction execute the active action.
// some of active-actions are flow-manager need to run.
func (h *activeflowHandler) executeAction(ctx context.Context, callID uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "executeAction",
			"call_id":   callID,
			"action_id": act.ID,
		},
	)

	// update current action in active-flow
	af, err := h.updateCurrentAction(ctx, callID, act)
	if err != nil {
		log.Errorf("Could not update the current action. err: %v", err)
		return nil, fmt.Errorf("could not update the current action. err: %v", err)
	}

	switch act.Type {
	case action.TypeAgentCall:
		if errHandle := h.actionHandleAgentCall(ctx, callID, af); errHandle != nil {
			log.Errorf("Could not handle the agent_call action correctly. err: %v", err)
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeBranch:
		if errHandle := h.actionHandleBranch(ctx, callID, af); errHandle != nil {
			log.Errorf("Could not handle the branch action correctly. err: %v", err)
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeConditionCallDigits:
		if errHandle := h.actionHandleConditionCallDigits(ctx, callID, af); errHandle != nil {
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeConditionCallStatus:
		if errHandle := h.actionHandleConditionCallStatus(ctx, callID, af); errHandle != nil {
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeConferenceJoin:
		if errHandle := h.actionHandleConferenceJoin(ctx, callID, af); errHandle != nil {
			log.Errorf("Could not handle the conference_join action correctly. err: %v", err)
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeConnect:
		if errHandle := h.actionHandleConnect(ctx, callID, af); errHandle != nil {
			log.Errorf("Could not handle the connect action correctly. err: %v", err)
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeGoto:
		if errHandle := h.actionHandleGoto(ctx, callID, af); errHandle != nil {
			log.Errorf("Could not handle the goto action correctly. err: %v", err)
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypePatch:
		if errHandle := h.actionHandlePatch(ctx, callID, af); errHandle != nil {
			log.Errorf("Could not handle the patch action correctly. err: %v", err)
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypePatchFlow:
		if errHandle := h.actionHandlePatchFlow(ctx, callID, af); errHandle != nil {
			log.Errorf("Could not handle the patch_flow action correctly. err: %v", err)
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeQueueJoin:
		if errHandle := h.actionHandleQueueJoin(ctx, callID, af); errHandle != nil {
			log.Errorf("Could not handle the queue_join action correctly. err: %v", err)
			return nil, err
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeTranscribeRecording:
		if err := h.actionHandleTranscribeRecording(ctx, af, callID, act); err != nil {
			log.Errorf("Could not handle the recording_to_text action correctly. err: %v", err)
			// we can move on to the next action even it's failed
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeTranscribeStart:
		if err := h.actionHandleTranscribeStart(ctx, af, callID, act); err != nil {
			log.Errorf("Could not start the transcribe. err: %v", err)
			// we can move on to the next action even it's failed
		}
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)
	}

	return act, nil
}

// updateCurrentAction updates the current action in active-flow.
// returns updated active flow
func (h *activeflowHandler) updateCurrentAction(ctx context.Context, callID uuid.UUID, act *action.Action) (*activeflow.ActiveFlow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "activeFlowUpdateCurrentAction",
			"call_id":   callID,
			"action_id": act,
		},
	)

	// get af
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return nil, err
	}

	// update active flow
	af.ExecutedActions = append(af.ExecutedActions, af.CurrentAction)
	af.CurrentAction = *act
	af.ForwardActionID = action.IDEmpty
	af.TMUpdate = dbhandler.GetCurTime()
	af.ExecuteCount++

	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active-flow's current action. err: %v", err)
		return nil, err
	}

	// get updated activeflow
	res, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		// because we
		log.Errorf("Could not get updated active flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveFlowUpdated, res)

	return res, err
}

// getNextAction returns next action from the active-flow
// It sets next action to current action.
func (h *activeflowHandler) getNextAction(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowGetNextAction",
		"call_id":           callID,
		"current_action_id": caID,
	})
	log.Debug("Getting next action.")

	// get active-flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
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
		resAction := h.actionHandler.CreateActionHangup()
		log.Warn("The actions are empty. Returning the action hangup.")
		return resAction, nil

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
		for _, act := range af.Actions {
			if act.ID == af.ForwardActionID {
				log.WithField("action", act).Debugf("Found move action.")
				return &act, nil
			}
		}
		log.WithField("actions", af.Actions).Errorf("Could not find forward action in the actions. forward_action_id: %v", af.ForwardActionID)
		return nil, fmt.Errorf("could not find move action in the actions array")
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
		// check if the no more actions left, return finishID here.
		log.Infof("No more action left. found: %v, idx: %v", found, idx)

		actionHangup := h.actionHandler.CreateActionHangup()
		return actionHangup, nil
	}

	res := af.Actions[idx+1]
	return &res, nil
}
