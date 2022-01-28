package flowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

const (
	maxActiveFlowExecuteCount = 100
)

// ActiveFlowCreate creates a active flow
func (h *flowHandler) ActiveFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error) {
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
	curTime := getCurTime()
	tmpAF := &activeflow.ActiveFlow{
		CallID:     callID,
		FlowID:     flowID,
		CustomerID:     f.CustomerID,
		WebhookURI: f.WebhookURI,

		CurrentAction: action.Action{
			ID: action.IDStart,
		},
		ExecuteCount:    0,
		ForwardActionID: action.IDEmpty,

		Actions: f.Actions,

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
	h.notifyHandler.PublishWebhookEvent(ctx, activeflow.EventTypeActiveFlowCreated, af.WebhookURI, af)

	return af, nil
}

// ActiveFlowSetForwardActionID sets the forward action id of the call.
func (h *flowHandler) ActiveFlowSetForwardActionID(ctx context.Context, callID uuid.UUID, actionID uuid.UUID, forwardNow bool) error {
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
func (h *flowHandler) ActiveFlowNextActionGet(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ActiveFlowNextActionGet",
		"call_id":           callID,
		"current_action_id": caID,
	})

	// get next action from the active
	nextAction, err := h.activeFlowGetNextAction(ctx, callID, caID)
	if err != nil {
		log.Errorf("Could not get next action. err: %v", err)
		return nil, err
	}
	log.WithField("action", nextAction).Debug("Found next action.")

	// execute the active action
	res, err := h.executeActiveAction(ctx, callID, nextAction)
	if err != nil {
		log.Errorf("Could not execute the active action. err: %v", err)
		return nil, err
	}

	return res, nil
}

// executeActiveAction execute the active action.
// some of active-actions are flow-manager need to run.
func (h *flowHandler) executeActiveAction(ctx context.Context, callID uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "executeActiveAction",
			"call_id":   callID,
			"action_id": act.ID,
		},
	)

	// update current action in active-flow
	if err := h.activeFlowUpdateCurrentAction(ctx, callID, act); err != nil {
		log.Errorf("Could not update the current action. err: %v", err)
		return nil, fmt.Errorf("could not update the current action. err: %v", err)
	}

	switch act.Type {
	case action.TypeAgentCall:
		if err := h.activeFlowHandleActionAgentCall(ctx, callID, act); err != nil {
			log.Errorf("Could not handle the agent_call action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeConferenceJoin:
		execAct, err := h.activeFlowHandleActionConferenceJoin(ctx, callID, act)
		if err != nil {
			log.Errorf("Could not handle the conference_join action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.executeActiveAction(ctx, callID, execAct)

	case action.TypeConnect:
		if err := h.activeFlowHandleActionConnect(ctx, callID, act); err != nil {
			log.Errorf("Could not handle the connect action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeGoto:
		execAct, err := h.activeFlowHandleActionGoto(ctx, callID, act)
		if err != nil {
			log.Errorf("Could not handle the goto action correctly. err: %v", err)
			return nil, err
		}

		return h.executeActiveAction(ctx, callID, execAct)

	case action.TypePatch:
		// handle the patch
		// add the patched actions to the active-flow
		execAct, err := h.activeFlowHandleActionPatch(ctx, callID, act)
		if err != nil {
			log.Errorf("Could not handle the patch action correctly. err: %v", err)
			return nil, err
		}

		// execute the updated action
		return h.executeActiveAction(ctx, callID, execAct)

	case action.TypePatchFlow:
		// handle the patch_flow
		// add the patched actions to the active-flow
		execAct, err := h.activeFlowHandleActionPatchFlow(ctx, callID, act)
		if err != nil {
			log.Errorf("Could not handle the patch_flow action correctly. err: %v", err)
			return nil, err
		}

		// execute the updated action
		return h.executeActiveAction(ctx, callID, execAct)

	case action.TypeQueueJoin:
		// handle the queue_join
		execAct, err := h.activeFlowHandleActionQueueJoin(ctx, callID, act)
		if err != nil {
			log.Errorf("Could not handle the queue_join action correctly. err: %v", err)
			return nil, err
		}

		// execute the updated action
		return h.executeActiveAction(ctx, callID, execAct)

	case action.TypeTranscribeRecording:
		if err := h.activeFlowHandleActionTranscribeRecording(ctx, callID, act); err != nil {
			log.Errorf("Could not handle the recording_to_text action correctly. err: %v", err)
			// we can move on to the next action even it's failed
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeTranscribeStart:
		if err := h.activeFlowHandleActionTranscribeStart(ctx, callID, act); err != nil {
			log.Errorf("Could not start the transcribe. err: %v", err)
			// we can move on to the next action even it's failed
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)
	}

	return act, nil
}

// activeFlowUpdateCurrentAction updates the current action in active-flow.
func (h *flowHandler) activeFlowUpdateCurrentAction(ctx context.Context, callID uuid.UUID, act *action.Action) error {
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
		return err
	}

	// update active flow
	af.CurrentAction = *act
	af.ForwardActionID = action.IDEmpty
	af.TMUpdate = getCurTime()
	af.ExecuteCount++

	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active-flow's current action. err: %v", err)
		return err
	}

	// get updated activeflow
	tmp, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		// because we
		log.Errorf("Could not get updated active flow. err: %v", err)
		return nil
	}
	h.notifyHandler.PublishWebhookEvent(ctx, activeflow.EventTypeActiveFlowUpdated, tmp.WebhookURI, tmp)

	return nil
}

// activeFlowGetNextAction returns next action from the active-flow
// It sets next action to current action.
func (h *flowHandler) activeFlowGetNextAction(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
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
		resAction := *(h.CreateActionHangup())
		log.Warn("The actions are empty. Returning the action hangup.")
		return &resAction, nil

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

	// check the move action id.
	if af.ForwardActionID != action.IDEmpty {
		log.Debug("The move action ID exist.")
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
	var nextAction action.Action
	if !found || idx >= (len(af.Actions)-1) {
		// check if the no more actions left, return finishID here.
		log.Infof("No more action left. found: %v, idx: %v", found, idx)

		// create finish hangup
		nextAction = *(h.CreateActionHangup())
	} else {
		nextAction = af.Actions[idx+1]
	}

	return &nextAction, nil
}
