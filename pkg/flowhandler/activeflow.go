package flowhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

const (
	maxActiveFlowExecuteCount = 100
)

// FlowCreate creates a flow
func (h *flowHandler) ActiveFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error) {

	// get flow
	f, err := h.db.FlowGet(ctx, flowID)
	if err != nil {
		logrus.Errorf("Could not get the flow. err: %v", err)
		return nil, err
	}

	// create activeflow
	curTime := getCurTime()
	tmpAF := &activeflow.ActiveFlow{
		CallID:     callID,
		FlowID:     flowID,
		UserID:     f.UserID,
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
		return nil, err
	}

	// get created active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		return nil, err
	}

	return af, nil
}

// ActiveFlowSetForwardActionID sets the move action id of the call.
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

	// check the action ID exists in the actions.
	found := false
	for _, a := range af.Actions {
		if a.ID == actionID {
			found = true
			break
		}
	}
	if !found {
		log.Errorf("Could not find move action id in the actions.")
		return fmt.Errorf("move action id not found")
	}

	// update active flow
	af.ForwardActionID = actionID
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow.")
		return err
	}

	// send action next
	if forwardNow {
		if err := h.reqHandler.CMV1CallActionNext(ctx, callID); err != nil {
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
			"func":    "executeActiveAction",
			"call_id": callID,
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
		if err := h.activeFlowHandleActionConferenceJoin(ctx, callID, act); err != nil {
			log.Errorf("Could not handle the conference_join action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypeConnect:
		if err := h.activeFlowHandleActionConnect(ctx, callID, act); err != nil {
			log.Errorf("Could not handle the connect action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.executeActiveAction(ctx, callID, act)

	case action.TypeGoto:
		act, err := h.activeFlowHandleActionGoto(ctx, callID, act)
		if err != nil {
			log.Errorf("Could not handle the goto action correctly. err: %v", err)
			return nil, err
		}

		return h.executeActiveAction(ctx, callID, act)

	case action.TypePatch:
		// handle the patch
		// add the patched actions to the active-flow
		if err := h.activeFlowHandleActionPatch(ctx, callID, act); err != nil {
			log.Errorf("Could not handle the patch action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

	case action.TypePatchFlow:
		// handle the patch
		// add the patched actions to the active-flow
		if err := h.activeFlowHandleActionPatchFlow(ctx, callID, act); err != nil {
			log.Errorf("Could not handle the patch_flow action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, act.ID)

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

// activeFlowHandleActionPatch handles action patch with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *flowHandler) activeFlowHandleActionPatch(ctx context.Context, callID uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":   callID,
		"action_id": act.ID,
	})

	// patch the actions from the remote
	patchedActions, err := h.actionPatchGet(act, callID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return err
	}

	// generate action id
	for _, act := range patchedActions {
		act.ID = uuid.Must(uuid.NewV4())
	}

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return err
	}

	// append the patched actions to the active flow
	if err := appendActionsAfterID(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionPatchFlow handles action patch_flow with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *flowHandler) activeFlowHandleActionPatchFlow(ctx context.Context, callID uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":   callID,
		"action_id": act.ID,
	})

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return err
	}

	// patch the actions from the remote
	patchedActions, err := h.actionPatchFlowGet(act, af.UserID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return err
	}

	// generate action id
	for _, act := range patchedActions {
		act.ID = uuid.Must(uuid.NewV4())
	}

	// append the patched actions to the active flow
	if err := appendActionsAfterID(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionPatchFlow handles action patch_flow with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *flowHandler) activeFlowHandleActionConferenceJoin(ctx context.Context, callID uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":   callID,
		"action_id": act.ID,
	})
	log.Debugf("Action detail. action: %v", act)

	var opt action.OptionConferenceJoin
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	conferenceID := uuid.FromStringOrNil(opt.ConferenceID)
	log = log.WithField("conference_id", conferenceID)

	// get conference
	conf, err := h.reqHandler.CFV1ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return err
	}
	if conf.Status != cfconference.StatusProgressing {
		log.Errorf("The conference is not ready. status: %s", conf.Status)
		return fmt.Errorf("conference is not ready")
	}

	// get flow
	f, err := h.FlowGet(ctx, conf.FlowID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return err
	}

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return err
	}

	// append the patched actions to the active flow
	if err := appendActionsAfterID(af, act.ID, f.Actions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionConnect handles action connect with active flow.
func (h *flowHandler) activeFlowHandleActionConnect(ctx context.Context, callID uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":   callID,
		"action_id": act.ID,
	})

	// get active-flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return fmt.Errorf("could not get active-flow. err: %v", err)
	}

	// create conference room for connect
	cf, err := h.reqHandler.CFV1ConferenceCreate(ctx, af.UserID, cfconference.TypeConnect, "", "", 86400, "", nil, nil, nil)
	if err != nil {
		log.Errorf("Could not create conference for connect. err: %v", err)
		return fmt.Errorf("could not create conference for connect. err: %v", err)
	}
	log = log.WithFields(logrus.Fields{
		"conference": cf.ID,
	})
	log.Debug("Created conference for connect.")

	// create a temp flow connect conference join
	optJoin := action.OptionConferenceJoin{
		ConferenceID: cf.ID.String(),
	}
	optString, err := json.Marshal(optJoin)
	if err != nil {
		log.Errorf("Could not marshal the conference join option. err: %v", err)
		return fmt.Errorf("could not marshal the conference join option. err: %v", err)
	}

	tmpCF := &flow.Flow{
		UserID:  cf.UserID,
		Persist: false,
		Actions: []action.Action{
			{
				Type:   action.TypeConferenceJoin,
				Option: optString,
			},
		},
	}

	var optConnect action.OptionConnect
	if err := json.Unmarshal(act.Option, &optConnect); err != nil {
		log.Errorf("Could not unmarshal the connect option. err: %v", err)
		return fmt.Errorf("could not unmarshal the connect option. err: %v", err)
	}

	// create a flow
	connectCF, err := h.FlowCreate(ctx, tmpCF)
	if err != nil {
		log.Errorf("Could not create a temporary flow for connect. err: %v", err)
		return fmt.Errorf("could not create a call flow. err: %v", err)
	}

	// create a call for each destination
	successCount := 0
	for _, dest := range optConnect.Destinations {
		source := &address.Address{
			Type:   address.Type(optConnect.Source.Type),
			Target: optConnect.Source.Target,
			Name:   optConnect.Source.Name,
		}

		destination := &address.Address{
			Type:   address.Type(dest.Type),
			Target: dest.Target,
			Name:   dest.Name,
		}

		// create a call
		resCall, err := h.reqHandler.CMV1CallCreate(ctx, connectCF.UserID, connectCF.ID, source, destination)
		if err != nil {
			log.Errorf("Could not create a outgoing call for connect. err: %v", err)
			continue
		}

		// add the chained call id if the unchained option is false
		if !optConnect.Unchained {
			if err := h.reqHandler.CMV1CallAddChainedCall(ctx, callID, resCall.ID); err != nil {
				log.Warnf("Could not add the chained call id. Hangup the call. chained_call_id: %s", resCall.ID)
				_, errHangup := h.reqHandler.CMV1CallHangup(ctx, resCall.ID)
				if errHangup != nil {
					log.Errorf("Could not hangup the call. chained_call_id: %s, err: %v", resCall.ID, errHangup)
				}
				continue
			}
		}

		log.Debugf("Created outgoing call for connect. call: %s", resCall.ID)
		successCount++
	}

	if successCount == 0 {
		log.Errorf("Could not create any successful outgoingcall.")
		return fmt.Errorf("could not create any successful outgoing call")
	}

	// put original call into the created conference
	resAction := action.Action{
		ID:     uuid.Must(uuid.NewV4()),
		Type:   action.TypeConferenceJoin,
		Option: optString,
	}

	// add the created action next to the given action id.
	if err := appendActionsAfterID(af, act.ID, []action.Action{resAction}); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// update active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionGoto handles action goto with active flow.
func (h *flowHandler) activeFlowHandleActionGoto(ctx context.Context, callID uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "activeFlowHandleActionGoto",
		"call_id":   callID,
		"action_id": act.ID,
	})
	log.WithField("action", act).Debug("Handle action goto.")

	// get active-flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return nil, fmt.Errorf("could not get active-flow. err: %v", err)
	}

	var opt action.OptionGoto
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not marshal the goto action's option. err: %v", err)
		return nil, err
	}

	var res *action.Action
	targetIndex := -1
	for i, a := range af.Actions {
		if a.ID == opt.TargetID {
			targetIndex = i
			res = &a
			break
		}
	}
	if targetIndex == -1 {
		log.Errorf("Could not find the target ID in the action flow.")
		return nil, fmt.Errorf("no target id found")
	}

	if !opt.Loop {
		return res, nil
	} else if opt.LoopCount > 0 {
		if err := h.activeFlowHandleActionGotoUpdate(ctx, af, act); err != nil {
			log.Errorf("Could not update the active flow for action goto. err: %v", err)
			return nil, err
		}
		return res, nil
	} else {
		// get next action
		for i, a := range af.Actions {
			if a.ID == act.ID {
				if i+1 >= len(af.Actions) {
					log.Errorf("Exceed length of actions.")
					return nil, fmt.Errorf("exceed action length")
				}
				res = &af.Actions[i+1]
				return res, nil
			}
		}
		return nil, fmt.Errorf("action not found")
	}
}

// activeFlowHandleActionTranscribeRecording handles transcribe_recording
func (h *flowHandler) activeFlowHandleActionTranscribeRecording(ctx context.Context, callID uuid.UUID, act *action.Action) error {

	log := logrus.WithFields(logrus.Fields{
		"call_id":   callID,
		"action_id": act.ID,
	})

	var optRecordingToText action.OptionTranscribeRecording
	if err := json.Unmarshal(act.Option, &optRecordingToText); err != nil {
		log.Errorf("Could not unmarshal the recording_to_text option. err: %v", err)
		return err
	}

	// transcribe-recording
	if err := h.reqHandler.TSV1CallRecordingCreate(ctx, callID, optRecordingToText.Language, optRecordingToText.WebhookURI, optRecordingToText.WebhookMethod, 120, 30); err != nil {
		log.Errorf("Could not handle the call recording to text correctly. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionTranscribeStart handles transcribe_start
func (h *flowHandler) activeFlowHandleActionTranscribeStart(ctx context.Context, callID uuid.UUID, act *action.Action) error {

	log := logrus.WithFields(logrus.Fields{
		"call_id":   callID,
		"action_id": act.ID,
	})

	var opt action.OptionTranscribeStart
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}

	// transcribe-recording
	trans, err := h.reqHandler.TSV1StreamingCreate(ctx, callID, opt.Language, opt.WebhookURI, opt.WebhookMethod)
	if err != nil {
		log.Errorf("Could not handle the call recording to text correctly. err: %v", err)
		return err
	}

	log.Debugf("The streaming transcribe has started. transcribe: %v", trans)
	return nil
}

// activeFlowHandleActionAgentCall handles action agent_call with active flow.
func (h *flowHandler) activeFlowHandleActionAgentCall(ctx context.Context, callID uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "activeFlowHandleActionAgentCall",
		"call_id":   callID,
		"action_id": act.ID,
	})

	var opt action.OptionAgentCall
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	agentID := uuid.FromStringOrNil(opt.AgentID)
	log = log.WithField("agent_id", agentID)

	// get active-flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return fmt.Errorf("could not get active-flow. err: %v", err)
	}

	// create conference room for agent_call
	cf, err := h.reqHandler.CFV1ConferenceCreate(ctx, af.UserID, cfconference.TypeConnect, "", "", 86400, "", nil, nil, nil)
	if err != nil {
		log.Errorf("Could not create conference for agent_call. err: %v", err)
		return fmt.Errorf("could not create conference for agent_call. err: %v", err)
	}
	log = log.WithFields(logrus.Fields{
		"conference_id": cf.ID,
	})
	log.Debug("Created conference for agent_call.")

	// get call info
	c, err := h.reqHandler.CMV1CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	// call to the agent
	if errDial := h.reqHandler.AMV1AgentDial(ctx, agentID, &c.Source, cf.ConfbridgeID); errDial != nil {
		log.Errorf("Could not dial to the agent. err: %v", errDial)
		return errDial
	}

	// create action connect for conference join
	optJoin := action.OptionConferenceJoin{
		ConferenceID: cf.ID.String(),
	}
	optString, err := json.Marshal(optJoin)
	if err != nil {
		log.Errorf("Could not marshal the conference join option. err: %v", err)
		return fmt.Errorf("could not marshal the conference join option. err: %v", err)
	}

	resAction := action.Action{
		ID:     uuid.Must(uuid.NewV4()),
		Type:   action.TypeConferenceJoin,
		Option: optString,
	}

	// add the created action next to the given action id.
	if err := appendActionsAfterID(af, act.ID, []action.Action{resAction}); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// update active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

func appendActionsAfterID(af *activeflow.ActiveFlow, id uuid.UUID, act []action.Action) error {

	var res []action.Action

	// get idx
	idx := -1
	for i, act := range af.Actions {
		if act.ID == id {
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

// activeFlowHandleActionGotoUpdate Updates the action flow.
// it updates the loop_count.
func (h *flowHandler) activeFlowHandleActionGotoUpdate(ctx context.Context, af *activeflow.ActiveFlow, act *action.Action) error {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":      "activeFlowHandleActionGotoLoopUpdate",
			"call_id":   af.CallID,
			"action_id": act.ID,
		},
	)

	// find goto action
	for i, a := range af.Actions {
		if a.ID == act.ID {
			tmpGoto := af.Actions[i]
			var tmpGotoOpt action.OptionGoto
			if err := json.Unmarshal(tmpGoto.Option, &tmpGotoOpt); err != nil {
				log.Errorf("Could not unmarshal the goto option. err: %v", err)
				return err
			}

			tmpGotoOpt.LoopCount--
			tmpRaw, err := json.Marshal(tmpGotoOpt)
			if err != nil {
				log.Errorf("Could not marshal the goto option. err: %v", err)
				return err
			}
			af.Actions[i].Option = tmpRaw

			// update active flow
			if err := h.db.ActiveFlowSet(ctx, af); err != nil {
				log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
				return err
			}

			break
		}
	}

	return nil
}
