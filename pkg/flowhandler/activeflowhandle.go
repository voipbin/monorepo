package flowhandler

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
func (h *flowHandler) getActionsFromFlow(flowID uuid.UUID, customerID uuid.UUID) ([]action.Action, error) {
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

	if f.CustomerID != customerID {
		log.Errorf("The customer has no permission. customer_id: %d", customerID)
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

// activeFlowHandleActionGotoLoop handles goto action's loop condition.
// it updates the loop_count.
func (h *flowHandler) activeFlowHandleActionGotoLoop(ctx context.Context, af *activeflow.ActiveFlow) error {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":              "activeFlowHandleActionGotoUpdate",
			"call_id":           af.CallID,
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
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionGotoNext handles goto action's no loop condition.
// it updates the loop_count.
func (h *flowHandler) activeFlowHandleActionGotoLoopStop(ctx context.Context, af *activeflow.ActiveFlow) error {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":              "activeFlowHandleActionGotoUpdate",
			"call_id":           af.CallID,
			"current_action_id": af.CurrentAction.ID,
		},
	)

	// find goto action
	idx := 0
	found := false
	for i, a := range af.Actions {
		if a.ID == af.CurrentAction.ID {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("current action not found")
	}

	if idx+1 >= len(af.Actions) {
		return fmt.Errorf("out of action range")
	}

	targetAction := af.Actions[idx+1]
	af.ForwardActionID = targetAction.ID
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionPatch handles action patch with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *flowHandler) activeFlowHandleActionPatch(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionPatch",
		"call_id":           callID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	// patch the actions from the remote
	patchedActions, err := h.actionPatchGet(act, callID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return err
	}

	tmpActions, err := h.generateFlowActions(ctx, patchedActions)
	if err != nil {
		log.Errorf("Could not generate flow actions. err: %v", err)
		return err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, tmpActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	af.ForwardActionID = tmpActions[0].ID
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionPatchFlow handles action patch_flow with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *flowHandler) activeFlowHandleActionPatchFlow(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionPatchFlow",
		"call_id":           callID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	var option action.OptionPatchFlow
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}

	// patch the actions from the remote
	patchedActions, err := h.getActionsFromFlow(option.FlowID, af.CustomerID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return err
	}

	// generate action id
	for _, act := range patchedActions {
		act.ID = uuid.Must(uuid.NewV4())
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	af.ForwardActionID = patchedActions[0].ID
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionConditionDigits handles action condition_digits with active flow.
// it checks the received digits and sets the forward action id.
func (h *flowHandler) activeFlowHandleActionConditionDigits(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionConditionDigits",
		"call_id":           callID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction

	var opt action.OptionConditionDigits
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}
	log.WithField("option", opt).Debugf("Detail option.")

	// gets the received digits
	digits, err := h.reqHandler.CMV1CallGetDigits(ctx, callID)
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
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionConferenceJoin handles action conference_join with active flow.
// it gets the given conference's flow and replace it.
func (h *flowHandler) activeFlowHandleActionConferenceJoin(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionConferenceJoin",
		"call_id":           callID,
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
	f, err := h.FlowGet(ctx, conf.FlowID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return err
	}

	// replace the patched actions to the active flow
	if err := replaceActions(af, act.ID, f.Actions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	af.ForwardActionID = f.Actions[0].ID
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionConnect handles action connect with active flow.
func (h *flowHandler) activeFlowHandleActionConnect(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionConnect",
		"call_id":           callID,
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
	connectCF, err := h.FlowCreate(ctx, cf.CustomerID, flow.TypeFlow, "", "", false, actions)
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
	masterCallID := callID
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
	af.TMUpdate = getCurTime()

	// update active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionGoto handles action goto with active flow.
func (h *flowHandler) activeFlowHandleActionGoto(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionGoto",
		"call_id":           callID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := &af.CurrentAction
	log.WithField("action", act).Debug("Handle action goto.")

	var opt action.OptionGoto
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not marshal the goto action's option. err: %v", err)
		return err
	}

	if !opt.Loop {
		af.ForwardActionID = opt.TargetID
		if err := h.db.ActiveFlowSet(ctx, af); err != nil {
			log.Errorf("Could not update the active flow. err: %v", err)
			return err
		}
		return nil
	}

	if opt.LoopCount > 0 {
		if err := h.activeFlowHandleActionGotoLoop(ctx, af); err != nil {
			log.Errorf("Could not update the active flow for action goto. err: %v", err)
			return err
		}
		return nil
	}

	// loop count is 0. no more loop.
	if err := h.activeFlowHandleActionGotoLoopStop(ctx, af); err != nil {
		log.Errorf("Could not update the active flow for action goto no loop. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionTranscribeRecording handles transcribe_recording
func (h *flowHandler) activeFlowHandleActionTranscribeRecording(ctx context.Context, af *activeflow.ActiveFlow, callID uuid.UUID, act *action.Action) error {
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
	res, err := h.reqHandler.TSV1CallRecordingCreate(ctx, af.CustomerID, callID, optRecordingToText.Language, 120000, 30)
	if err != nil {
		log.Errorf("Could not handle the call recording to text correctly. err: %v", err)
		return err
	}
	log.WithField("transcribes", res).Debugf("Received transcribes.")

	return nil
}

// activeFlowHandleActionTranscribeStart handles transcribe_start
func (h *flowHandler) activeFlowHandleActionTranscribeStart(ctx context.Context, af *activeflow.ActiveFlow, callID uuid.UUID, act *action.Action) error {
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
	trans, err := h.reqHandler.TSV1StreamingCreate(ctx, af.CustomerID, callID, tstranscribe.TypeCall, opt.Language)
	if err != nil {
		log.Errorf("Could not handle the call recording to text correctly. err: %v", err)
		return err
	}

	log.Debugf("The streaming transcribe has started. transcribe: %v", trans)
	return nil
}

// activeFlowHandleActionAgentCall handles action agent_call with active flow.
func (h *flowHandler) activeFlowHandleActionAgentCall(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionAgentCall",
		"call_id":           callID,
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
	c, err := h.reqHandler.CMV1CallGet(ctx, callID)
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
	log.Debugf("Dialing to the agent. call_id: %s, flow_id: %s", callID, f.ID)
	agentDial, err := h.reqHandler.AMV1AgentDial(ctx, opt.AgentID, &c.Source, f.ID, callID)
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
	af.TMUpdate = getCurTime()

	// update active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionQueueJoin handles queue_join action type.
func (h *flowHandler) activeFlowHandleActionQueueJoin(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionQueueJoin",
		"call_id":           callID,
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
	qc, err := h.reqHandler.QMV1QueueCreateQueuecall(ctx, q.ID, qmqueuecall.ReferenceTypeCall, callID, exitActionID)
	if err != nil {
		log.WithField("exit_action_id", exitActionID).Errorf("Could not create the queuecall. Forward to the exit action. err: %v", err)
		errForward := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, callID, exitActionID, true)
		if errForward != nil {
			log.Errorf("Could not forward the active flow. err: %v", errForward)
		}
	}
	log.WithField("queuecall", qc).Debug("Created the queuecall.")

	// get flow's action
	patchedActions, err := h.getActionsFromFlow(qc.FlowID, q.CustomerID)
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
	af.TMUpdate = getCurTime()
	af.ForwardActionID = patchedActions[0].ID
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
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

// activeFlowHandleActionBranch handles branch action type.
func (h *flowHandler) activeFlowHandleActionBranch(ctx context.Context, callID uuid.UUID, af *activeflow.ActiveFlow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "activeFlowHandleActionBranch",
		"call_id":           callID,
		"current_action_id": af.CurrentAction.ID,
	})
	act := af.CurrentAction

	var opt action.OptionBranch
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the branch option. err: %v", err)
		return err
	}

	// get received digits
	digits, err := h.reqHandler.CMV1CallGetDigits(ctx, callID)
	if err != nil {
		log.Errorf("Could not get digits. err: %v", err)
		return err
	}

	targetID, ok := opt.TargetIDs[digits]
	if !ok {
		targetID = opt.DefaultID
		log.Debugf("Input digit is not listed in the branch. digit: %s, default_target_id: %s", digits, targetID)
	}

	_, err = h.getActionFromActions(af.Actions, targetID)
	if err != nil {
		log.Errorf("Could not find target action. err: %v", err)
		return err
	}

	af.ForwardActionID = targetID
	if errSet := h.db.ActiveFlowSet(ctx, af); errSet != nil {
		log.Errorf("Could not update the active flow. err: %v", errSet)
		return err
	}

	return nil
}
