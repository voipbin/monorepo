package flowhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
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
func (h *flowHandler) getActionsFromFlow(flowID uuid.UUID, userID uint64) ([]action.Action, error) {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"func": "getActionsFromFlow",
		},
	)

	// get flow
	f, err := h.db.FlowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get flow info. err: %v", err)
		return nil, err
	}

	if f.UserID != userID {
		log.Errorf("The user has no permission. user_id: %d", userID)
		return nil, fmt.Errorf("no flow found")
	}

	return f.Actions, nil
}

// getExitActionID returns exit action id
func (h *flowHandler) getExitActionID(actions []action.Action, actionID uuid.UUID) (uuid.UUID, error) {
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

// activeFlowHandleActionGotoUpdate Updates the action flow.
// it updates the loop_count.
func (h *flowHandler) activeFlowHandleActionGotoUpdate(ctx context.Context, af *activeflow.ActiveFlow, act *action.Action) error {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":      "activeFlowHandleActionGotoUpdate",
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

// activeFlowHandleActionPatch handles action patch with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *flowHandler) activeFlowHandleActionPatch(ctx context.Context, callID uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "activeFlowHandleActionPatch",
		"call_id":   callID,
		"action_id": act.ID,
	})

	// patch the actions from the remote
	patchedActions, err := h.actionPatchGet(act, callID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return nil, err
	}

	// generate action id
	for _, act := range patchedActions {
		act.ID = uuid.Must(uuid.NewV4())
	}

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return nil, err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return nil, fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return nil, err
	}

	return &patchedActions[0], nil
}

// activeFlowHandleActionPatchFlow handles action patch_flow with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *flowHandler) activeFlowHandleActionPatchFlow(ctx context.Context, callID uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "activeFlowHandleActionPatchFlow",
		"call_id":   callID,
		"action_id": act.ID,
	})

	var option action.OptionPatchFlow
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return nil, err
	}

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return nil, err
	}

	// patch the actions from the remote
	patchedActions, err := h.getActionsFromFlow(option.FlowID, af.UserID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return nil, err
	}

	// generate action id
	for _, act := range patchedActions {
		act.ID = uuid.Must(uuid.NewV4())
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return nil, fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return nil, err
	}

	return &patchedActions[0], nil
}

// activeFlowHandleActionConferenceJoin handles action conference_join with active flow.
// it gets the given conference's flow and replace it.
func (h *flowHandler) activeFlowHandleActionConferenceJoin(ctx context.Context, callID uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "activeFlowHandleActionConferenceJoin",
		"call_id":   callID,
		"action_id": act.ID,
	})
	log.Debugf("Action detail. action: %v", act)

	var opt action.OptionConferenceJoin
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return nil, err
	}
	conferenceID := uuid.FromStringOrNil(opt.ConferenceID)
	log = log.WithField("conference_id", conferenceID)

	// get conference
	conf, err := h.reqHandler.CFV1ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return nil, err
	}
	if conf.Status != cfconference.StatusProgressing {
		log.Errorf("The conference is not ready. status: %s", conf.Status)
		return nil, fmt.Errorf("conference is not ready")
	}

	// get flow
	f, err := h.FlowGet(ctx, conf.FlowID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return nil, err
	}

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return nil, err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, f.Actions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return nil, fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return nil, err
	}

	return &f.Actions[0], nil
}

// activeFlowHandleActionConnect handles action connect with active flow.
func (h *flowHandler) activeFlowHandleActionConnect(ctx context.Context, callID uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "activeFlowHandleActionConnect",
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
		"conference_id": cf.ID,
	})
	log.WithField("conference", cf).Debug("Created conference for connect.")

	// create a temp flow connect conference join
	optJoin := action.OptionConferenceJoin{
		ConferenceID: cf.ID.String(),
	}
	optString, err := json.Marshal(optJoin)
	if err != nil {
		log.Errorf("Could not marshal the conference join option. err: %v", err)
		return fmt.Errorf("could not marshal the conference join option. err: %v", err)
	}

	actions := []action.Action{
		{
			Type:   action.TypeConferenceJoin,
			Option: optString,
		},
	}

	// create a flow
	connectCF, err := h.FlowCreate(ctx, cf.UserID, flow.TypeFlow, "", "", false, "", actions)
	if err != nil {
		log.Errorf("Could not create a temporary flow for connect. err: %v", err)
		return fmt.Errorf("could not create a call flow. err: %v", err)
	}

	var optConnect action.OptionConnect
	if err := json.Unmarshal(act.Option, &optConnect); err != nil {
		log.Errorf("Could not unmarshal the connect option. err: %v", err)
		return fmt.Errorf("could not unmarshal the connect option. err: %v", err)
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

	// append the created action to the given action id.
	if err := appendActions(af, act.ID, []action.Action{resAction}); err != nil {
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
		"func":      "activeFlowHandleActionTranscribeRecording",
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
		"func":      "activeFlowHandleActionTranscribeStart",
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
	log.WithField("call", c).Debug("Found call info.")

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

	// append the created action to the given action id.
	if err := appendActions(af, act.ID, []action.Action{resAction}); err != nil {
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

// activeFlowHandleActionQueueJoin handles queue_join action type.
func (h *flowHandler) activeFlowHandleActionQueueJoin(ctx context.Context, callID uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "activeFlowHandleActionQueueJoin",
		"call_id":   callID,
		"action_id": act.ID,
	})

	var opt action.OptionQueueJoin
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return nil, err
	}
	queueID := opt.QueueID
	log = log.WithField("queue_id", queueID)

	// get queue info
	q, err := h.reqHandler.QMV1QueueGet(ctx, queueID)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", q).Debug("Found queue info.")

	// get flow's action
	patchedActions, err := h.getActionsFromFlow(q.FlowID, q.UserID)
	if err != nil {
		log.Errorf("Could not get queue flow's actions. err: %v", err)
		return nil, err
	}

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return nil, err
	}

	// get exit action id
	// because we will update the active flow's actions, we need to getting the exit action id before.
	exitActionID, err := h.getExitActionID(af.Actions, act.ID)
	if err != nil {
		log.Errorf("Could not get exit action id. err: %v", err)
		return nil, err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not replace new action. err: %v", err)
		return nil, fmt.Errorf("could not replace new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after replace the patched actions. err: %v", err)
		return nil, err
	}

	// send the queue join request
	qc, err := h.reqHandler.QMV1QueueCreateQueuecall(ctx, q.ID, qmqueuecall.ReferenceTypeCall, callID, exitActionID)
	if err != nil {
		log.WithField("exit_action_id", exitActionID).Errorf("Could not create the queuecall. Forward to the exit action. err: %v", err)
		errForward := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, callID, exitActionID, true)
		if errForward != nil {
			log.Errorf("Could not forward the active flow. err: %v", errForward)
		}
	}
	log.WithField("queuecall", qc).Debug("Created the queuecall.")

	return &patchedActions[0], nil
}
