package activeflowhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	tstranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

// actionHandleGotoLoop handles goto action's loop condition.
// it updates the loop_count.
func (h *activeflowHandler) actionHandleGotoLoop(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":              "activeFlowHandleActionGotoLoop",
			"activeflow_id":     af.ID,
			"reference_type":    af.ReferenceType,
			"reference_id":      af.ReferenceID,
			"current_action_id": af.CurrentAction.ID,
		},
	)

	// find goto action
	idx := 0
	var act action.Action
	found := false
	for i, a := range af.Actions {
		if a.ID == af.CurrentAction.ID {
			idx = i
			act = a
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("current action not found")
	}

	var opt action.OptionGoto
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the goto option. err: %v", err)
		return err
	}

	opt.LoopCount--
	raw, err := json.Marshal(opt)
	if err != nil {
		log.Errorf("Could not marshal the goto option. err: %v", err)
		return err
	}
	af.Actions[idx].Option = raw

	af.ForwardActionID = opt.TargetID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandlePatch handles action patch with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *activeflowHandler) actionHandlePatch(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionPatch",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	// patch the actions from the remote
	patchedActions, err := h.actionHandler.ActionPatchGet(act, id)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = dbhandler.GetCurTime()

	// set active flow
	af.ForwardActionID = patchedActions[0].ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandlePatchFlow handles action patch_flow with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *activeflowHandler) actionHandlePatchFlow(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionPatchFlow",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	var option action.OptionPatchFlow
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}

	// patch the actions from the remote
	patchedActions, err := h.getActionsFromFlow(ctx, option.FlowID, af.CustomerID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = dbhandler.GetCurTime()

	// set active flow
	af.ForwardActionID = patchedActions[0].ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleConditionCallDigits handles action condition_call_digits with active flow.
// it checks the received digits and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionCallDigits(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "actionHandleConditionCallDigits",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	var opt action.OptionConditionCallDigits
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}
	log.WithField("option", opt).Debugf("Detail option.")

	// gets the received digits
	digits, err := h.reqHandler.CMV1CallGetDigits(ctx, id)
	if err != nil {
		log.Errorf("Could not get digits. err: %v", err)
		return err
	}
	log.Debugf("Received digits. digits: %s", digits)

	// check the conditions
	if opt.Length != 0 && len(digits) >= opt.Length {
		log.Debugf("Condition matched length. len: %d", opt.Length)
		return nil
	} else if opt.Key != "" && strings.Contains(digits, opt.Key) {
		log.Debugf("Condition matched key. key: %s", opt.Key)
		return nil
	}

	// failed
	log.Debugf("Could not match the condition. Move to the false target. false_target_id: %s", opt.FalseTargetID)
	af.ForwardActionID = opt.FalseTargetID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleConditionCallStatus handles action condition_call_status with active flow.
// it checks the call's status and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionCallStatus(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "actionHandleConditionCallStatus",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	var opt action.OptionConditionCallStatus
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}
	log.WithField("option", opt).Debugf("Detail option.")

	// gets the call
	c, err := h.reqHandler.CMV1CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get call. err: %v", err)
		return err
	}
	log.WithField("call", c).Debugf("Received call info. call_id: %s", id)

	// match the condition
	if string(opt.Status) == string(c.Status) {
		log.Debugf("Condtion matched status. status: %s", opt.Status)
		return nil
	}

	// failed
	log.Debugf("Could not match the condition. Move to the false target. false_target_id: %s", opt.FalseTargetID)
	af.ForwardActionID = opt.FalseTargetID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleConferenceJoin handles action conference_join with active flow.
// it gets the given conference's flow and replace it.
func (h *activeflowHandler) actionHandleConferenceJoin(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionConferenceJoin",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	log.Debugf("Action detail. action: %v", act)

	var opt action.OptionConferenceJoin
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	log = log.WithField("conference_id", opt.ConferenceID)

	// get conference
	conf, err := h.reqHandler.CFV1ConferenceGet(ctx, opt.ConferenceID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return err
	}
	if conf.Status != cfconference.StatusProgressing {
		log.Errorf("The conference is not ready. status: %s", conf.Status)
		return fmt.Errorf("conference is not ready")
	}

	// get flow
	f, err := h.reqHandler.FMV1FlowGet(ctx, conf.FlowID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, f.Actions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = dbhandler.GetCurTime()

	// set active flow
	af.ForwardActionID = f.Actions[0].ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleConnect handles action connect with active flow.
func (h *activeflowHandler) actionHandleConnect(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionConnect",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	// create conference room for connect
	cf, err := h.reqHandler.CFV1ConferenceCreate(ctx, af.CustomerID, cfconference.TypeConnect, "", "", 86400, nil, nil, nil)
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
		ConferenceID: cf.ID,
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
	connectCF, err := h.reqHandler.FMV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "", "", actions, false)
	if err != nil {
		log.Errorf("Could not create a temporary flow for connect. err: %v", err)
		return fmt.Errorf("could not create a call flow. err: %v", err)
	}

	var optConnect action.OptionConnect
	if err := json.Unmarshal(act.Option, &optConnect); err != nil {
		log.Errorf("Could not unmarshal the connect option. err: %v", err)
		return fmt.Errorf("could not unmarshal the connect option. err: %v", err)
	}

	// set master call id.
	masterCallID := id
	if optConnect.Unchained {
		masterCallID = uuid.Nil
	}

	// create a call
	resCall, err := h.reqHandler.CMV1CallsCreate(ctx, connectCF.CustomerID, connectCF.ID, masterCallID, &optConnect.Source, optConnect.Destinations)
	if err != nil {
		log.Errorf("Could not create a outgoing call for connect. err: %v", err)
		return err
	}
	log.WithField("calls", resCall).Debugf("Created outgoing call for connect without master call id. count: %d", len(resCall))

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
	af.TMUpdate = dbhandler.GetCurTime()

	// update active flow
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleGoto handles action goto with active flow.
func (h *activeflowHandler) actionHandleGoto(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionGoto",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction
	log.WithField("action", act).Debug("Handle action goto.")

	var opt action.OptionGoto
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not marshal the goto action's option. err: %v", err)
		return err
	}

	if opt.LoopCount <= 0 {
		log.Debugf("Loop over. Move to the next action. loop_count: %d", opt.LoopCount)
		return nil
	}

	if err := h.actionHandleGotoLoop(ctx, af); err != nil {
		log.Errorf("Could not update the active flow for action goto. err: %v", err)
		return err
	}
	return nil
}

// actionHandleTranscribeRecording handles transcribe_recording
func (h *activeflowHandler) actionHandleTranscribeRecording(ctx context.Context, af *activeflow.Activeflow, id uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "activeFlowHandleActionTranscribeRecording",
		"activeflow_id":  id,
		"reference_type": af.ReferenceType,
		"reference_id":   af.ReferenceID,
		"action_id":      act.ID,
	})

	var optRecordingToText action.OptionTranscribeRecording
	if err := json.Unmarshal(act.Option, &optRecordingToText); err != nil {
		log.Errorf("Could not unmarshal the recording_to_text option. err: %v", err)
		return err
	}

	// transcribe-recording
	res, err := h.reqHandler.TSV1CallRecordingCreate(ctx, af.CustomerID, id, optRecordingToText.Language, 120000, 30)
	if err != nil {
		log.Errorf("Could not handle the call recording to text correctly. err: %v", err)
		return err
	}
	log.WithField("transcribes", res).Debugf("Received transcribes.")

	return nil
}

// actionHandleTranscribeStart handles transcribe_start
func (h *activeflowHandler) actionHandleTranscribeStart(ctx context.Context, af *activeflow.Activeflow, id uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "activeFlowHandleActionTranscribeStart",
		"activeflow_id":  id,
		"reference_type": af.ReferenceType,
		"reference_id":   af.ReferenceID,
		"action_id":      act.ID,
	})

	var opt action.OptionTranscribeStart
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}

	// transcribe-recording
	trans, err := h.reqHandler.TSV1StreamingCreate(ctx, af.CustomerID, id, tstranscribe.TypeCall, opt.Language)
	if err != nil {
		log.Errorf("Could not handle the call recording to text correctly. err: %v", err)
		return err
	}

	log.Debugf("The streaming transcribe has started. transcribe: %v", trans)
	return nil
}

// actionHandleAgentCall handles action agent_call with active flow.
func (h *activeflowHandler) actionHandleAgentCall(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionAgentCall",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	var opt action.OptionAgentCall
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	log = log.WithField("agent_id", opt.AgentID)

	// create conference room for agent_call
	cf, err := h.reqHandler.CFV1ConferenceCreate(ctx, af.CustomerID, cfconference.TypeConnect, "", "", 86400, nil, nil, nil)
	if err != nil {
		log.Errorf("Could not create conference for agent_call. err: %v", err)
		return fmt.Errorf("could not create conference for agent_call. err: %v", err)
	}
	log = log.WithFields(logrus.Fields{
		"conference_id": cf.ID,
	})
	log.Debug("Created conference for agent_call.")

	// get call info
	c, err := h.reqHandler.CMV1CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}
	log.WithField("call", c).Debug("Found call info.")

	// generate the flow for the agent call
	f, err := h.generateFlowForAgentCall(ctx, c.CustomerID, cf.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		return err
	}
	log.WithField("flow", f).Debug("Created a flow.")

	// call to the agent
	log.Debugf("Dialing to the agent. call_id: %s, flow_id: %s", id, f.ID)
	agentDial, err := h.reqHandler.AMV1AgentDial(ctx, opt.AgentID, &c.Source, f.ID, id)
	if err != nil {
		log.Errorf("Could not dial to the agent. err: %v", err)
		return err
	}
	log.WithField("agent_dial", agentDial).Debugf("Created agent_dial. agent_dial_id: %s", agentDial.ID)

	// create action connect for conference join
	optJoin := action.OptionConferenceJoin{
		ConferenceID: cf.ID,
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
	af.TMUpdate = dbhandler.GetCurTime()

	// update active flow
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleQueueJoin handles queue_join action type.
func (h *activeflowHandler) actionHandleQueueJoin(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionQueueJoin",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	var opt action.OptionQueueJoin
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	queueID := opt.QueueID
	log = log.WithField("queue_id", queueID)

	// get queue info
	q, err := h.reqHandler.QMV1QueueGet(ctx, queueID)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
		return err
	}
	log.WithField("queue", q).Debug("Found queue info.")

	// get exit action id
	// because we will update the active flow's actions, we need to getting the exit action id before.
	exitActionID, err := h.getExitActionID(af.Actions, act.ID)
	if err != nil {
		log.Errorf("Could not get exit action id. err: %v", err)
		return err
	}

	// send the queue join request
	qc, err := h.reqHandler.QMV1QueueCreateQueuecall(ctx, q.ID, qmqueuecall.ReferenceTypeCall, id, exitActionID)
	if err != nil {
		log.WithField("exit_action_id", exitActionID).Errorf("Could not create the queuecall. Forward to the exit action. err: %v", err)
		errForward := h.reqHandler.FMV1ActiveflowUpdateForwardActionID(ctx, id, exitActionID, true)
		if errForward != nil {
			log.Errorf("Could not forward the active flow. err: %v", errForward)
		}
	}
	log.WithField("queuecall", qc).Debug("Created the queuecall.")

	// get flow's action
	patchedActions, err := h.getActionsFromFlow(ctx, qc.FlowID, q.CustomerID)
	if err != nil {
		log.Errorf("Could not get queue flow's actions. err: %v", err)
		return err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not replace new action. err: %v", err)
		return fmt.Errorf("could not replace new action. err: %v", err)
	}

	// set active flow
	af.TMUpdate = dbhandler.GetCurTime()
	af.ForwardActionID = patchedActions[0].ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after replace the patched actions. err: %v", err)
		return err
	}

	// send the queuecall excute request
	tmp, err := h.reqHandler.QMV1QueuecallExecute(ctx, qc.ID, 1000)
	if err != nil {
		log.Errorf("Could not send the execute request. err: %v", err)
		return err
	}
	log.WithField("queuecall", tmp).Debug("Queuecall executed.")

	return nil
}

// actionHandleBranch handles branch action type.
func (h *activeflowHandler) actionHandleBranch(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionBranch",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := af.CurrentAction

	var opt action.OptionBranch
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the branch option. err: %v", err)
		return err
	}

	// get received digits
	digits, err := h.reqHandler.CMV1CallGetDigits(ctx, id)
	if err != nil {
		log.Errorf("Could not get digits. err: %v", err)
		return err
	}

	// send digits reset
	if errDigits := h.reqHandler.CMV1CallSetDigits(ctx, id, ""); errDigits != nil {
		// we got the error here, but this is minor issue.
		// just write the log.
		log.Errorf("Could not reset the call digits. err: %v", errDigits)
	}

	targetID, ok := opt.TargetIDs[digits]
	if !ok {
		targetID = opt.DefaultTargetID
		log.Debugf("Input digit is not listed in the branch. digit: %s, default_target_id: %s", digits, targetID)
	}

	af.ForwardActionID = targetID
	if errSet := h.db.ActiveflowUpdate(ctx, af); errSet != nil {
		log.Errorf("Could not update the active flow. err: %v", errSet)
		return err
	}

	return nil
}

// actionHandleMessageSend handles message_send action type.
func (h *activeflowHandler) actionHandleMessageSend(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "actionHandleMessageSend",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := af.CurrentAction

	var opt action.OptionMessageSend
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the message_send option. err: %v", err)
		return err
	}

	tmp, err := h.reqHandler.MMV1MessageSend(ctx, af.CustomerID, opt.Source, opt.Destinations, opt.Text)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return err
	}
	log.WithField("message", tmp).Debugf("Send the message correctly. message_id: %s", tmp.ID)

	return nil
}

// actionHandleCall handles action call with active flow.
func (h *activeflowHandler) actionHandleCall(ctx context.Context, id uuid.UUID, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "actionHandleCall",
		"activeflow_id":     id,
		"reference_type":    af.ReferenceType,
		"reference_id":      af.ReferenceID,
		"current_action_id": af.CurrentAction.ID,
	})
	log.Debugf("Executing the action call. reference_id: %s", af.ReferenceID)

	act := &af.CurrentAction

	var opt action.OptionCall
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return fmt.Errorf("could not unmarshal the option. err: %v", err)
	}

	flowID := opt.FlowID
	if flowID == uuid.Nil {
		// create a flow
		tmpFlow, err := h.reqHandler.FMV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "", "", opt.Actions, false)
		if err != nil {
			log.Errorf("Could not create a temporary flow for connect. err: %v", err)
			return fmt.Errorf("could not create a call flow. err: %v", err)
		}
		log.WithField("flow", tmpFlow).Debugf("Created a temp flow. flow_id: %s", tmpFlow.ID)

		flowID = tmpFlow.ID
	}

	masterCallID := uuid.Nil
	if opt.Chained && af.ReferenceType == activeflow.ReferenceTypeCall {
		masterCallID = af.ReferenceID
	}

	resCalls, err := h.reqHandler.CMV1CallsCreate(ctx, af.CustomerID, flowID, masterCallID, opt.Source, opt.Destinations)
	if err != nil {
		log.Errorf("Could not create a outgoing call for connect. err: %v", err)
		return err
	}
	log.WithField("calls", resCalls).Debugf("Created outgoing calls for action call. count: %d", len(resCalls))

	return nil
}
