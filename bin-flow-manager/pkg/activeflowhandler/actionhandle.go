package activeflowhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"

	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	conversationmedia "monorepo/bin-conversation-manager/models/media"

	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
)

// actionHandleGotoLoop handles goto action's loop condition.
// it updates the loop_count.
func (h *activeflowHandler) actionHandleGotoLoop(ctx context.Context, af *activeflow.Activeflow, act *action.Action, opt *action.OptionGoto) error {
	log := logrus.New().WithFields(logrus.Fields{
		"func":       "actionHandleGotoLoop",
		"activeflow": af,
		"action":     act,
		"option":     opt,
	})

	// find action
	_, orgAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, act.ID, false)
	if err != nil {
		log.Errorf("Could not get original action. err: %v", err)
		return err
	}

	// find goto action
	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, opt.TargetID, false)
	if err != nil {
		log.Errorf("Could not find loop target action. err: %v", err)
		return err
	}

	opt.LoopCount--
	raw, err := json.Marshal(opt)
	if err != nil {
		log.Errorf("Could not marshal the goto option. err: %v", err)
		return err
	}

	orgAction.Option = raw
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleFetch handles action patch with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *activeflowHandler) actionHandleFetch(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleFetch",
		"activeflow": af,
	})

	act := &af.CurrentAction

	// patch the actions from the remote
	fetchedActions, err := h.actionHandler.ActionFetchGet(act, af.ID, af.ReferenceID)
	if err != nil {
		log.Errorf("Could not fetch the actions from the remote. err: %v", err)
		return err
	}

	// push the actions
	if errPush := h.PushStack(ctx, af, uuid.Nil, fetchedActions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errPush
	}

	return nil
}

// actionHandleFetchFlow handles action patch_flow with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *activeflowHandler) actionHandleFetchFlow(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleFetchFlow",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var option action.OptionFetchFlow
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}

	// patch the actions from the remote
	fetchedActions, err := h.getActionsFromFlow(ctx, option.FlowID, af.CustomerID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return err
	}

	// push the actions
	if errPush := h.PushStack(ctx, af, uuid.Nil, fetchedActions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errPush
	}

	return nil
}

// actionHandleConditionCallDigits handles action condition_call_digits with active flow.
// it checks the received digits and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionCallDigits(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleConditionCallDigits",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionConditionCallDigits
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}
	log.WithField("option", opt).Debugf("Detail option.")

	// gets the received digits
	digits, err := h.reqHandler.CallV1CallGetDigits(ctx, af.ReferenceID)
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

	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, opt.FalseTargetID, false)
	if err != nil {
		log.Errorf("Could not find false target action. err: %v", err)
		return err
	}

	// failed
	log.Debugf("Could not match the condition. Move to the false target. false_target_id: %s", opt.FalseTargetID)
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleConditionCallStatus handles action condition_call_status with active flow.
// it checks the call's status and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionCallStatus(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleConditionCallStatus",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionConditionCallStatus
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}
	log.WithField("option", opt).Debugf("Detail option.")

	// gets the call
	c, err := h.reqHandler.CallV1CallGet(ctx, af.ReferenceID)
	if err != nil {
		log.Errorf("Could not get call. err: %v", err)
		return err
	}
	log.WithField("call", c).Debugf("Received call info. call_id: %s", c.ID)

	// match the condition
	if string(opt.Status) == string(c.Status) {
		log.Debugf("Condtion matched status. status: %s", opt.Status)
		return nil
	}

	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, opt.FalseTargetID, false)
	if err != nil {
		log.Errorf("Could not find false target action. err: %v", err)
		return err
	}
	log.Debugf("Could not match the condition. Forward to the false target. target_stack_id:%s, target_action_id: %s", targetStackID, targetAction.ID)

	// failed
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleConditionDatetime handles action condition_datetime with activeflow.
// it checks the current datetime and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionDatetime(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleConditionDatetime",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionConditionDatetime
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}
	log.WithField("option", opt).Debugf("Detail option.")

	// get current time
	current := time.Now().UTC()
	log.Debugf("Current time. datetime: %s", current.String())

	match := true

	// check the weekdays
	// if the option has weekdays, we need to check this first.
	if len(opt.Weekdays) != 0 {
		match = false
		weekday := int(current.Weekday())
		for _, day := range opt.Weekdays {
			if day == weekday {
				match = true
				break
			}
		}

		if !match {
			log.Debugf("The weekday does not match. weekdays: %v, current: %d", opt.Weekdays, weekday)
		}
	}

	// check month
	if match {
		if opt.Month > 0 {
			res := compareCondition(opt.Condition, opt.Month, int(current.Month()))
			if !res {
				match = false
			}
		}

		if !match {
			log.Debugf("The month does not match. condition: %s, month: %d, current: %d", opt.Condition, opt.Month, current.Month())
		}
	}

	// check day
	if match {
		if opt.Day > 0 {
			res := compareCondition(opt.Condition, opt.Day, current.Day())
			if !res {
				match = false
			}
		}

		if !match {
			log.Debugf("The day does not match. condition: %s, day: %d, current: %d", opt.Condition, opt.Day, current.Day())
		}
	}

	// check hour
	if match {
		if opt.Hour >= 0 {
			res := compareCondition(opt.Condition, opt.Hour, current.Hour())
			if !res {
				match = false
			}
		}

		if !match {
			log.Debugf("The hour does not match. condition: %s, hour: %d, current: %d", opt.Condition, opt.Hour, current.Hour())
		}
	}

	// check minute
	if match {
		if opt.Minute >= 0 {
			res := compareCondition(opt.Condition, opt.Minute, current.Minute())
			if !res {
				match = false
			}
		}

		if !match {
			log.Debugf("The minute does not match. condition: %s, minute: %d, current: %d", opt.Condition, opt.Minute, current.Minute())
		}
	}

	// it matched all conditions.
	// nothing to do here.
	if match {
		return nil
	}

	// could not pass the match conditions.
	// gets the false target action
	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, opt.FalseTargetID, false)
	if err != nil {
		log.Errorf("Could not find false target action. err: %v", err)
		return err
	}

	// sets the false target action
	log.Debugf("Could not match the condition. Move to the false target. false_target_id: %s", opt.FalseTargetID)
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleConditionVariable handles action condition_variable with activeflow.
// it checks the given variable and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionVariable(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleConditionVariable",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionConditionVariable
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}
	log.WithField("option", opt).Debugf("Detail option.")

	match := false
	switch opt.ValueType {
	case action.OptionConditionVariableTypeString:
		match = compareCondition(opt.Condition, opt.Variable, opt.ValueString)

	case action.OptionConditionVariableTypeNumber:
		tmp, err := strconv.ParseFloat(opt.Variable, 32)
		if err != nil {
			log.Errorf("Could not parse the variable. err: %v", err)
			break
		}
		match = compareCondition(opt.Condition, float32(tmp), opt.ValueNumber)

	case action.OptionConditionVariableTypeLength:
		match = compareCondition(opt.Condition, len(opt.Variable), opt.ValueLength)
	}

	if match {
		return nil
	}

	// could not pass the match conditions.
	// gets the false target action
	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, opt.FalseTargetID, false)
	if err != nil {
		log.Errorf("Could not find false target action. err: %v", err)
		return err
	}

	// sets the false target action
	log.Debugf("Could not match the condition. Move to the false target. false_target_id: %s", opt.FalseTargetID)
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// actionHandleConferenceJoin handles action conference_join with active flow.
// it gets the given conference's flow and replace it.
func (h *activeflowHandler) actionHandleConferenceJoin(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleConferenceJoin",
		"activeflow": af,
	})
	act := &af.CurrentAction

	log.Debugf("Action detail. action: %v", act)

	var opt action.OptionConferenceJoin
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	log = log.WithField("conference_id", opt.ConferenceID)

	if af.ReferenceType != activeflow.ReferenceTypeCall {
		log.Errorf("Wrong type of reference. Only reference type call is supported. reference_type: %s", af.ReferenceType)
		return fmt.Errorf("wrong reference type. reference_type: %s", af.ReferenceType)
	}

	sv, err := h.reqHandler.ConferenceV1ServiceTypeConferencecallStart(ctx, opt.ConferenceID, cfconferencecall.ReferenceTypeCall, af.ReferenceID)
	if err != nil {
		log.Errorf("Could not start the service. err: %v", err)
		return errors.Wrap(err, "Could not start the service.")
	}

	// push the actions
	if errPush := h.PushStack(ctx, af, sv.ID, sv.PushActions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errPush
	}

	return nil
}

// actionHandleConnect handles action connect with active flow.
func (h *activeflowHandler) actionHandleConnect(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleConnect",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionConnect
	if errMarshal := json.Unmarshal(act.Option, &opt); errMarshal != nil {
		log.Errorf("Could not unmarshal the connect option. err: %v", errMarshal)
		return fmt.Errorf("could not unmarshal the connect option. err: %v", errMarshal)
	}

	// create a confbridge for connect
	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, af.CustomerID, cmconfbridge.TypeConnect)
	if err != nil {
		log.Errorf("Could not create a confbridge for connect. err: %v", err)
		return errors.Wrap(err, "could not create a confbridge for connect")
	}
	log = log.WithFields(logrus.Fields{
		"confbridge_id": cb.ID,
	})
	log.WithField("confbridge", cb).Debug("Created confbridge for connect.")

	// create a temp flow connect confbridge join
	tmpOpt := action.OptionConfbridgeJoin{
		ConfbridgeID: cb.ID,
	}
	tmpOptString, err := json.Marshal(tmpOpt)
	if err != nil {
		log.Errorf("Could not marshal the confbridge join option. err: %v", err)
		return fmt.Errorf("could not marshal the confbridge join option. err: %v", err)
	}
	tmpActions := []action.Action{
		{
			Type:   action.TypeConfbridgeJoin,
			Option: tmpOptString,
		},
		{
			Type: action.TypeHangup,
		},
	}

	// create a flow for connect call
	f, err := h.reqHandler.FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "tmp", "tmp flow for action connect", tmpActions, false)
	if err != nil {
		log.Errorf("Could not create a temporary flow for connect. err: %v", err)
		return fmt.Errorf("could not create a call flow. err: %v", err)
	}

	// get call create options
	var earlyExecution bool
	var executeNext bool
	if opt.EarlyMedia {
		earlyExecution = true
		executeNext = false
	} else {
		earlyExecution = false
		if len(opt.Destinations) > 1 {
			// we are making more than 1 call. so it's not
			executeNext = false
		} else {
			executeNext = true
		}
	}

	// create a call for connect
	resCalls, resGroupcalls, err := h.reqHandler.CallV1CallsCreate(ctx, f.CustomerID, f.ID, af.ReferenceID, &opt.Source, opt.Destinations, earlyExecution, executeNext)
	if err != nil {
		log.Errorf("Could not create a outgoing call for connect. err: %v", err)
		return err
	}
	log.WithFields(logrus.Fields{
		"calls":      resCalls,
		"groupcalls": resGroupcalls,
	}).Debugf("Created outgoing calls for connect. call_count: %d, groupcall_count: %d", len(resCalls), len(resGroupcalls))

	if len(resCalls) == 0 && len(resGroupcalls) == 0 {
		log.WithFields(logrus.Fields{
			"calls":      resCalls,
			"groupcalls": resGroupcalls,
		}).Errorf("Could not create any outgoing calls or groupcalls for connect.")
		return fmt.Errorf("could not create any outgoing calls")
	}

	// create push actions for activeflow
	// put original call into the created conference
	pushActions := []action.Action{
		{
			ID:     h.utilHandler.UUIDCreate(),
			Type:   action.TypeConfbridgeJoin,
			Option: tmpOptString,
		},
	}

	if len(resGroupcalls) == 0 && opt.RelayReason {
		// get reference id
		// we consider the first call of get the reference
		referenceID := resCalls[0].ID
		log.Debugf("The connect action has relay reason option enabled. Adding the hangup relay action. reference_id: %s", referenceID)

		optHangup := action.OptionHangup{
			ReferenceID: referenceID,
		}
		optStringHangup, err := json.Marshal(optHangup)
		if err != nil {
			log.Errorf("Could not marshal the confbridge join option. err: %v", err)
			return fmt.Errorf("could not marshal the confbridge join option. err: %v", err)
		}

		actionHangupRelay := action.Action{
			ID:     h.utilHandler.UUIDCreate(),
			Type:   action.TypeHangup,
			Option: optStringHangup,
		}
		pushActions = append(pushActions, actionHangupRelay)
	}

	// push the actions
	if errPush := h.PushStack(ctx, af, uuid.Nil, pushActions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errPush
	}

	return nil
}

// actionHandleGoto handles action goto with active flow.
func (h *activeflowHandler) actionHandleGoto(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleGoto",
		"activeflow": af,
	})
	log.WithField("action", af.CurrentAction).Debug("Handle action goto.")

	act := &af.CurrentAction

	var opt action.OptionGoto
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not marshal the goto action's option. err: %v", err)
		return err
	}

	if opt.LoopCount <= 0 {
		log.Debugf("Loop over. Move to the next action. loop_count: %d", opt.LoopCount)
		return nil
	}

	if err := h.actionHandleGotoLoop(ctx, af, act, &opt); err != nil {
		log.Errorf("Could not update the active flow for action goto. err: %v", err)
		return err
	}
	return nil
}

// actionHandleTranscribeRecording handles transcribe_recording
func (h *activeflowHandler) actionHandleTranscribeRecording(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleTranscribeRecording",
		"activeflow": af,
	})

	act := &af.CurrentAction

	var optRecordingToText action.OptionTranscribeRecording
	if err := json.Unmarshal(act.Option, &optRecordingToText); err != nil {
		log.Errorf("Could not unmarshal the recording_to_text option. err: %v", err)
		return err
	}

	if af.ReferenceType != activeflow.ReferenceTypeCall {
		// nothing to do.
		log.Errorf("Invalid reference type. Currently, support the call type only. reference_type: %s", af.ReferenceType)
		return nil
	}

	c, err := h.reqHandler.CallV1CallGet(ctx, af.ReferenceID)
	if err != nil {
		log.Errorf("Could not get the call. err: %s", err)
		return err
	}

	// transcribe the recordings
	for _, recordingID := range c.RecordingIDs {
		tmp, err := h.reqHandler.TranscribeV1TranscribeStart(ctx, af.CustomerID, tmtranscribe.ReferenceTypeRecording, recordingID, optRecordingToText.Language, tmtranscribe.DirectionBoth)
		if err != nil {
			log.Errorf("Could not handle the call recording to text correctly. err: %v", err)
			return err
		}
		log.WithField("transcribes", tmp).Debugf("Transcribed the recording. transcribe_id: %s, recording_id: %s", tmp.ID, recordingID)
	}

	return nil
}

// actionHandleTranscribeStart handles transcribe_start
func (h *activeflowHandler) actionHandleTranscribeStart(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleTranscribeStart",
		"activeflow": af,
	})

	act := &af.CurrentAction

	if af.ReferenceType != activeflow.ReferenceTypeCall {
		log.Errorf("Unsupported reference type. reference_type: %s", af.ReferenceType)
		return nil
	}

	var opt action.OptionTranscribeStart
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}

	// transcribe start
	trans, err := h.reqHandler.TranscribeV1TranscribeStart(ctx, af.CustomerID, tmtranscribe.ReferenceTypeCall, af.ReferenceID, opt.Language, tmtranscribe.DirectionBoth)
	if err != nil {
		log.Errorf("Could not handle the call recording to text correctly. err: %v", err)
		return err
	}

	log.Debugf("The streaming transcribe has started. transcribe: %v", trans)
	return nil
}

// actionHandleQueueJoin handles queue_join action type.
func (h *activeflowHandler) actionHandleQueueJoin(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleQueueJoin",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionQueueJoin
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	log = log.WithField("queue_id", opt.QueueID)

	// get exit action info
	exitStackID, exitAction := h.stackmapHandler.GetNextAction(af.StackMap, af.CurrentStackID, &af.CurrentAction, false)
	log.WithField("exit_action", exitAction).Debugf("Found exit action info. stack_id: %s, action_id: %s", exitStackID, exitAction.ID)

	sv, err := h.reqHandler.QueueV1ServiceTypeQueuecallStart(ctx, opt.QueueID, af.ID, qmqueuecall.ReferenceTypeCall, af.ReferenceID, exitAction.ID)
	if err != nil {
		log.Errorf("Could not start the service. err: %v", err)
		return errors.Wrap(err, "Could not start the service.")
	}

	// push the actions
	if errPush := h.PushStack(ctx, af, sv.ID, sv.PushActions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errPush
	}

	// update the queuecall status to waiting.
	// updating the status to waiting after push the stack is important.
	// becasue in case of execute the exit action immediately after status set to waiting
	// it is possible to causing the race condition.
	// to avoid the race condition, we have change the status after push the stack.
	_, err = h.reqHandler.QueueV1QueuecallUpdateStatusWaiting(ctx, sv.ID)
	if err != nil {
		log.Errorf("Could not update the queuecall status to waiting. err: %v", err)
		return errors.Wrap(err, "Could not update the queuecall status to waiting.")
	}

	return nil
}

// actionHandleBranch handles branch action type.
func (h *activeflowHandler) actionHandleBranch(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleBranch",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionBranch
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the branch option. err: %v", err)
		return err
	}

	// get variable
	v, err := h.variableHandler.Get(ctx, af.ID)
	if err != nil {
		log.Errorf("Could not get variable. err: %v", err)
		return err
	}

	// get target variable
	tmpVar := opt.Variable
	if tmpVar == "" {
		log.Debugf("The option has no variable. Setting a default. variable: %s", action.OptionBranchVariableDefault)
		tmpVar = action.OptionBranchVariableDefault
	}
	targetVar := v.Variables[tmpVar]

	// reset the variable
	variables := map[string]string{
		tmpVar: "",
	}
	log.Debugf("Resetting a variable. variable: %s", tmpVar)
	if errVar := h.variableHandler.SetVariable(ctx, af.ID, variables); errVar != nil {
		log.Errorf("Could not reset the variable. But Keep going the flow. err: %v", err)
	}

	targetID, ok := opt.TargetIDs[targetVar]
	if !ok {
		targetID = opt.DefaultTargetID
		log.Debugf("Input digit is not listed in the branch. variable: %s, variable_value: %s, default_target_id: %s", tmpVar, targetVar, targetID)
	}

	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, targetID, false)
	if err != nil {
		log.Errorf("Could not get target action. err: %v", err)
		return err
	}

	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if errSet := h.db.ActiveflowUpdate(ctx, af); errSet != nil {
		log.Errorf("Could not update the active flow. err: %v", errSet)
		return err
	}

	return nil
}

// actionHandleMessageSend handles message_send action type.
func (h *activeflowHandler) actionHandleMessageSend(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleMessageSend",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionMessageSend
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the message_send option. err: %v", err)
		return err
	}

	// send message
	tmp, err := h.reqHandler.MessageV1MessageSend(ctx, uuid.Nil, af.CustomerID, opt.Source, opt.Destinations, opt.Text)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return err
	}
	log.WithField("message", tmp).Debugf("Send the message correctly. message_id: %s", tmp.ID)

	return nil
}

// actionHandleCall handles action call with active flow.
func (h *activeflowHandler) actionHandleCall(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleCall",
		"activeflow": af,
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
		tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "", "", opt.Actions, false)
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

	resCalls, resGroupcalls, err := h.reqHandler.CallV1CallsCreate(ctx, af.CustomerID, flowID, masterCallID, opt.Source, opt.Destinations, opt.EarlyExecution, false)
	if err != nil {
		log.Errorf("Could not create a outgoing call for connect. err: %v", err)
		return err
	}
	log.WithFields(logrus.Fields{
		"calls":      resCalls,
		"groupcalls": resGroupcalls,
	}).Debugf("Created outgoing calls for action call. call_count: %d, groupcall_count: %d", len(resCalls), len(resGroupcalls))

	return nil
}

// actionHandleVariableSet handles action variable_set with active flow.
func (h *activeflowHandler) actionHandleVariableSet(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleVariableSet",
		"activeflow": af,
	})
	log.Debugf("Executing the action variable_set. reference_id: %s", af.ReferenceID)

	act := &af.CurrentAction

	var opt action.OptionVariableSet
	if errUnmarshal := json.Unmarshal(act.Option, &opt); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the option. err: %v", errUnmarshal)
		return fmt.Errorf("could not unmarshal the option. err: %v", errUnmarshal)
	}

	variables := map[string]string{
		opt.Key: opt.Value,
	}
	if errVariable := h.variableHandler.SetVariable(ctx, af.ID, variables); errVariable != nil {
		return fmt.Errorf("could not set varialbe. err: %v", errVariable)
	}

	return nil
}

// actionHandleWebhookSend handles action webhook_send with active flow.
func (h *activeflowHandler) actionHandleWebhookSend(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleWebhookSend",
		"activeflow": af,
	})
	log.Debugf("Executing the action webhook_send. reference_id: %s", af.ReferenceID)

	act := &af.CurrentAction

	var opt action.OptionWebhookSend
	if errUnmarshal := json.Unmarshal(act.Option, &opt); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the option. err: %v", errUnmarshal)
		return fmt.Errorf("could not unmarshal the option. err: %v", errUnmarshal)
	}
	log.Debugf("Sending webhook message. message: %s", opt.Data)

	if opt.Sync {
		if errSend := h.reqHandler.WebhookV1WebhookSendToDestination(ctx, af.CustomerID, opt.URI, wmwebhook.MethodType(opt.Method), wmwebhook.DataType(opt.DataType), []byte(opt.Data)); errSend != nil {
			log.Errorf("Could not send the webhook correctly on sync mode. err: %v", errSend)
		}
	} else {
		go func() {
			if errSend := h.reqHandler.WebhookV1WebhookSendToDestination(ctx, af.CustomerID, opt.URI, wmwebhook.MethodType(opt.Method), wmwebhook.DataType(opt.DataType), []byte(opt.Data)); errSend != nil {
				log.Errorf("Could not send the webhook correctlyon async mode. err: %v", errSend)
			}
		}()
	}

	return nil
}

// actionHandleConversationSend handles conversation_send action type.
func (h *activeflowHandler) actionHandleConversationSend(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleConversationSend",
		"activeflow": af,
	})
	act := &af.CurrentAction

	var opt action.OptionConversationSend
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the conversation_send option. err: %v", err)
		return err
	}

	// send a request
	if opt.Sync {
		res, err := h.reqHandler.ConversationV1MessageSend(ctx, opt.ConversationID, opt.Text, []conversationmedia.Media{})
		if err != nil {
			log.Errorf("Could not send the conversation_send request. err: %v", err)
			return err
		}
		log.WithField("message", res).Debugf("Sent the conversation message correctly. message_id: %s", res.ID)
	} else {
		go func() {
			res, err := h.reqHandler.ConversationV1MessageSend(ctx, opt.ConversationID, opt.Text, []conversationmedia.Media{})
			if err != nil {
				log.Errorf("Could not send the conversation_send request. err: %v", err)
				return
			}
			log.WithField("message", res).Debugf("Sent the conversation message correctly. message_id: %s", res.ID)
		}()
	}

	return nil
}

// actionHandleChatbotTalk handles action chatbot_talk with activeflow.
// it starts chatbot talk service.
func (h *activeflowHandler) actionHandleChatbotTalk(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleChatbotTalk",
		"activeflow": af,
	})
	act := &af.CurrentAction

	log.Debugf("Action detail. action: %v", act)

	var opt action.OptionChatbotTalk
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}

	if af.ReferenceType != activeflow.ReferenceTypeCall {
		log.Errorf("Wrong type of reference. Only reference type call is supported. reference_type: %s", af.ReferenceType)
		return fmt.Errorf("wrong reference type. reference_type: %s", af.ReferenceType)
	}

	// start service
	sv, err := h.reqHandler.ChatbotV1ServiceTypeChabotcallStart(ctx, opt.ChatbotID, af.ID, cbchatbotcall.ReferenceTypeCall, af.ReferenceID, opt.Gender, opt.Language, 3000)
	if err != nil {
		log.Errorf("Could not start the service. err: %v", err)
		return errors.Wrap(err, "Could not start the service.")
	}
	log.WithField("service", sv).Debugf("Started service. service_type: %s, service_id: %s", sv.Type, sv.ID)

	// push the actions
	if errPush := h.PushStack(ctx, af, sv.ID, sv.PushActions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errPush
	}

	return nil
}

// actionHandleStop handles action stop with activeflow.
func (h *activeflowHandler) actionHandleStop(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "actionHandleStop",
		"activeflow": af,
	})
	log.WithField("action", af.CurrentAction).Debug("Handle action stop.")

	actions := []action.Action{
		action.ActionFinish,
	}

	// push the actions
	if errPush := h.PushStack(ctx, af, uuid.Nil, actions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errPush
	}

	return nil
}
