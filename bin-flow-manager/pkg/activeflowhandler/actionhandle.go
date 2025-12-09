package activeflowhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	amaicall "monorepo/bin-ai-manager/models/aicall"

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
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleGotoLoop",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	// find action
	_, orgAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, act.ID, false)
	if err != nil {
		return errors.Wrapf(err, "could not get the original action.")
	}

	// find goto action
	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, opt.TargetID, false)
	if err != nil {
		return errors.Wrapf(err, "could not find the goto target action.")
	}

	opt.LoopCount--
	raw, err := json.Marshal(opt)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the goto option.")
	}

	if errUnmarshal := json.Unmarshal(raw, &orgAction.Option); errUnmarshal != nil {
		return errors.Wrapf(errUnmarshal, "could not unmarshal the option.")
	}
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.updateStackProgress(ctx, af); err != nil {
		return errors.Wrapf(err, "could not update the active flow after appending the patched actions.")
	}

	return nil
}

// actionHandleFetch handles action patch with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *activeflowHandler) actionHandleFetch(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleFetch",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	// patch the actions from the remote
	fetchedActions, err := h.actionHandler.ActionFetchGet(act, af.ID, af.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not fetch the actions from the remote.")
	}

	// push the actions
	if errPush := h.PushStack(ctx, af, uuid.Nil, fetchedActions); errPush != nil {
		return errors.Wrapf(errPush, "could not push the actions to the stack.")
	}

	return nil
}

// actionHandleFetchFlow handles action patch_flow with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *activeflowHandler) actionHandleFetchFlow(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleFetchFlow",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmp, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option.")
	}

	var option action.OptionFetchFlow
	if err := json.Unmarshal(tmp, &option); err != nil {
		return errors.Wrapf(err, "could not unmarshal the option.")
	}

	// patch the actions from the flow
	fetchedActions, err := h.actionGetsFromFlow(ctx, option.FlowID, af.CustomerID)
	if err != nil {
		return errors.Wrapf(err, "could not get actions from the flow. flow_id: %s", option.FlowID)
	}

	// push the actions
	if errPush := h.PushStack(ctx, af, uuid.Nil, fetchedActions); errPush != nil {
		return errors.Wrapf(errPush, "could not push the actions to the stack")
	}

	return nil
}

// actionHandleConditionCallDigits handles action condition_call_digits with active flow.
// it checks the received digits and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionCallDigits(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleConditionCallDigits",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmp, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option.")
	}

	var opt action.OptionConditionCallDigits
	if err := json.Unmarshal(tmp, &opt); err != nil {
		return errors.Wrapf(err, "could not unmarshal the option.")
	}
	log.WithField("option", opt).Debugf("Detail option.")

	// gets the received digits
	digits, err := h.reqHandler.CallV1CallGetDigits(ctx, af.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get digits.")
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
		return errors.Wrapf(err, "could not find false target action.")
	}

	// failed
	log.Debugf("Could not match the condition. Move to the false target. false_target_id: %s", opt.FalseTargetID)
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.updateStackProgress(ctx, af); err != nil {
		return errors.Wrapf(err, "could not update the active flow after appending the patched actions.")
	}

	return nil
}

// actionHandleConditionCallStatus handles action condition_call_status with active flow.
// it checks the call's status and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionCallStatus(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleConditionCallStatus",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmp, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option.")
	}

	var opt action.OptionConditionCallStatus
	if err := json.Unmarshal(tmp, &opt); err != nil {
		return errors.Wrapf(err, "could not unmarshal the option.")
	}
	log.WithField("option", opt).Debugf("Detail option.")

	// gets the call
	c, err := h.reqHandler.CallV1CallGet(ctx, af.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get call.")
	}
	log.WithField("call", c).Debugf("Received call info. call_id: %s", c.ID)

	// match the condition
	if string(opt.Status) == string(c.Status) {
		log.Debugf("Condtion matched status. status: %s", opt.Status)
		return nil
	}

	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, opt.FalseTargetID, false)
	if err != nil {
		return errors.Wrapf(err, "could not find false target action.")
	}
	log.Debugf("Could not match the condition. Forward to the false target. target_stack_id:%s, target_action_id: %s", targetStackID, targetAction.ID)

	// failed
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.updateStackProgress(ctx, af); err != nil {
		return errors.Wrapf(err, "could not update the activeflow after appended the patched actions.")
	}

	return nil
}

// actionHandleConditionDatetime handles action condition_datetime with activeflow.
// it checks the current datetime and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionDatetime(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleConditionDatetime",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmp, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option.")
	}

	var opt action.OptionConditionDatetime
	if err := json.Unmarshal(tmp, &opt); err != nil {
		return errors.Wrapf(err, "could not unmarshal the option.")
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
		return errors.Wrapf(err, "could not find false target action.")
	}

	// sets the false target action
	log.Debugf("Could not match the condition. Move to the false target. false_target_id: %s", opt.FalseTargetID)
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.updateStackProgress(ctx, af); err != nil {
		return errors.Wrapf(err, "could not update the active flow after appending the patched actions.")
	}

	return nil
}

// actionHandleConditionVariable handles action condition_variable with activeflow.
// it checks the given variable and sets the forward action id.
func (h *activeflowHandler) actionHandleConditionVariable(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleConditionVariable",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmp, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionConditionVariable
	if err := json.Unmarshal(tmp, &opt); err != nil {
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
	if err := h.updateStackProgress(ctx, af); err != nil {
		return errors.Wrapf(err, "could not update the active flow after appending the patched actions")
	}

	return nil
}

// actionHandleConferenceJoin handles action conference_join with active flow.
// it gets the given conference's flow and replace it.
func (h *activeflowHandler) actionHandleConferenceJoin(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleConferenceJoin",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	log.Debugf("Action detail. action: %v", act)

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionConferenceJoin
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	log = log.WithField("conference_id", opt.ConferenceID)

	if af.ReferenceType != activeflow.ReferenceTypeCall {
		log.Errorf("Wrong type of reference. Only reference type call is supported. reference_type: %s", af.ReferenceType)
		return fmt.Errorf("wrong reference type. reference_type: %s", af.ReferenceType)
	}

	sv, err := h.reqHandler.ConferenceV1ServiceTypeConferencecallStart(ctx, af.ID, opt.ConferenceID, cfconferencecall.ReferenceTypeCall, af.ReferenceID)
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
		"func":          "actionHandleConnect",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionConnect
	if errMarshal := json.Unmarshal(tmpOption, &opt); errMarshal != nil {
		log.Errorf("Could not unmarshal the connect option. err: %v", errMarshal)
		return fmt.Errorf("could not unmarshal the connect option. err: %v", errMarshal)
	}

	// create a confbridge for connect
	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, af.CustomerID, af.ID, cmconfbridge.ReferenceTypeCall, af.ReferenceID, cmconfbridge.TypeConnect)
	if err != nil {
		log.Errorf("Could not create a confbridge for connect. err: %v", err)
		return errors.Wrap(err, "could not create a confbridge for connect")
	}
	log = log.WithFields(logrus.Fields{
		"confbridge_id": cb.ID,
	})
	log.WithField("confbridge", cb).Debug("Created confbridge for connect.")

	tmpActions := []action.Action{
		{
			Type: action.TypeConfbridgeJoin,
			Option: action.ConvertOption(action.OptionConfbridgeJoin{
				ConfbridgeID: cb.ID,
			}),
		},
		{
			Type: action.TypeHangup,
		},
	}

	// create a flow for connect call
	f, err := h.reqHandler.FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "tmp", "tmp flow for action connect", tmpActions, uuid.Nil, false)
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
			ID:   h.utilHandler.UUIDCreate(),
			Type: action.TypeConfbridgeJoin,
			Option: action.ConvertOption(action.OptionConfbridgeJoin{
				ConfbridgeID: cb.ID,
			}),
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
			ID:   h.utilHandler.UUIDCreate(),
			Type: action.TypeHangup,
		}
		if errUnmarshal := json.Unmarshal(optStringHangup, &actionHangupRelay.Option); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal the option. err: %v", errUnmarshal)
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
		"func":          "actionHandleGoto",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionGoto
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
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
		"func":          "actionHandleTranscribeRecording",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionTranscribeRecording
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		return errors.Wrapf(err, "could not unmarshal the transcribe_recording option. err: %v", err)
	}

	if af.ReferenceType != activeflow.ReferenceTypeCall {
		// nothing to do.
		log.Errorf("Invalid reference type. Currently, support the call type only. reference_type: %s", af.ReferenceType)
		return nil
	}

	c, err := h.reqHandler.CallV1CallGet(ctx, af.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get the call. err: %v", err)
	}

	// transcribe the recordings
	for _, recordingID := range c.RecordingIDs {
		tmp, err := h.reqHandler.TranscribeV1TranscribeStart(
			ctx,
			af.CustomerID,
			af.ID,
			opt.OnEndFlowID,
			tmtranscribe.ReferenceTypeRecording,
			recordingID,
			opt.Language,
			tmtranscribe.DirectionBoth,
			30000,
		)
		if err != nil {
			return errors.Wrapf(err, "could not handle the call recording to text correctly. err: %v", err)
		}
		log.WithField("transcribes", tmp).Debugf("Transcribed the recording. transcribe_id: %s, recording_id: %s", tmp.ID, recordingID)
	}

	return nil
}

// actionHandleTranscribeStart handles transcribe_start
func (h *activeflowHandler) actionHandleTranscribeStart(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleTranscribeStart",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	if af.ReferenceType != activeflow.ReferenceTypeCall {
		log.Errorf("Unsupported reference type. reference_type: %s", af.ReferenceType)
		return nil
	}

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionTranscribeStart
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}

	// transcribe start
	trans, err := h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		af.CustomerID,
		af.ID,
		opt.OnEndFlowID,
		tmtranscribe.ReferenceTypeCall,
		af.ReferenceID,
		opt.Language,
		tmtranscribe.DirectionBoth,
		30000,
	)
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
		"func":          "actionHandleQueueJoin",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionQueueJoin
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}
	log = log.WithField("queue_id", opt.QueueID)

	sv, err := h.reqHandler.QueueV1ServiceTypeQueuecallStart(ctx, opt.QueueID, af.ID, qmqueuecall.ReferenceTypeCall, af.ReferenceID)
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
		"func":          "actionHandleBranch",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionBranch
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
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
	if errSet := h.updateStackProgress(ctx, af); errSet != nil {
		log.Errorf("Could not update the active flow. err: %v", errSet)
		return err
	}

	return nil
}

// actionHandleMessageSend handles message_send action type.
func (h *activeflowHandler) actionHandleMessageSend(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleMessageSend",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionMessageSend
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
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
		"func":          "actionHandleCall",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionCall
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return fmt.Errorf("could not unmarshal the option. err: %v", err)
	}

	flowID := opt.FlowID
	if flowID == uuid.Nil {
		// create a flow
		tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, af.CustomerID, flow.TypeFlow, "", "", opt.Actions, uuid.Nil, false)
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
		"func":          "actionHandleVariableSet",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionVariableSet
	if errUnmarshal := json.Unmarshal(tmpOption, &opt); errUnmarshal != nil {
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
		"func":          "actionHandleWebhookSend",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionWebhookSend
	if errUnmarshal := json.Unmarshal(tmpOption, &opt); errUnmarshal != nil {
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
		"func":          "actionHandleConversationSend",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionConversationSend
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
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

// actionHandleAITalk handles action ai_talk with activeflow.
// it starts ai talk service.
func (h *activeflowHandler) actionHandleAITalk(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleAITalk",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionAITalk
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}

	referenceType := amaicall.ReferenceTypeCall
	if af.ReferenceType == activeflow.ReferenceTypeConversation {
		referenceType = amaicall.ReferenceTypeConversation
	}

	// start service
	sv, err := h.reqHandler.AIV1ServiceTypeAIcallStart(ctx, opt.AIID, af.ID, referenceType, af.ReferenceID, opt.Resume, opt.Gender, opt.Language, 30000)
	if err != nil {
		return errors.Wrap(err, "Could not start the service.")
	}
	log.WithField("service", sv).Debugf("Started service. service_type: %s, service_id: %s", sv.Type, sv.ID)

	// push the actions
	if errPush := h.PushStack(ctx, af, sv.ID, sv.PushActions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errors.Wrapf(errPush, "Could not push the actions to the stack.")
	}

	return nil
}

// actionHandleStop handles action stop with activeflow.
func (h *activeflowHandler) actionHandleStop(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleStop",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

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

// actionHandleEmailSend handles action email_send with activeflow.
func (h *activeflowHandler) actionHandleEmailSend(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleEmailSend",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionEmailSend
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return err
	}

	// send message
	tmp, err := h.reqHandler.EmailV1EmailSend(ctx, af.CustomerID, af.ID, opt.Destinations, opt.Subject, opt.Content, opt.Attachments)
	if err != nil {
		log.Errorf("Could not send an email correctly. err: %v", err)
		return err
	}
	log.Debugf("Send an email correctly. email_id: %s", tmp.ID)

	return nil
}

// actionHandleAISummary handles action ai_summary with activeflow.
// it starts ai summary service.
func (h *activeflowHandler) actionHandleAISummary(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleAISummary",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionAISummary
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		return errors.Wrapf(err, "Could not unmarshal the option. err: %v", err)
	}

	// start service
	sv, err := h.reqHandler.AIV1ServiceTypeSummaryStart(
		ctx,
		af.CustomerID,
		af.ID,
		opt.OnEndFlowID,
		opt.ReferenceType,
		opt.ReferenceID,
		opt.Language,
		60000,
	)
	if err != nil {
		return errors.Wrap(err, "Could not start the service.")
	}
	log.WithField("service", sv).Debugf("Started service. service_type: %s, service_id: %s", sv.Type, sv.ID)

	// push the actions
	if errPush := h.PushStack(ctx, af, sv.ID, sv.PushActions); errPush != nil {
		return errors.Wrapf(errPush, "Could not push the actions to the stack.")
	}

	return nil
}

// actionHandleAITask handles action ai_task with activeflow.
// it starts ai task service.
func (h *activeflowHandler) actionHandleAITask(ctx context.Context, af *activeflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "actionHandleAITask",
		"activeflow_id": af.ID,
	})
	log.WithField("action", af.CurrentAction).Debugf("Executing action handle. type: %s, action_id: %s", af.CurrentAction.Type, af.CurrentAction.ID)

	act := &af.CurrentAction

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the option. err: %v", err)
	}

	var opt action.OptionAITask
	if err := json.Unmarshal(tmpOption, &opt); err != nil {
		log.Errorf("Could not unmarshal the transcribe_start option. err: %v", err)
		return err
	}

	referenceType := amaicall.ReferenceTypeCall
	if af.ReferenceType == activeflow.ReferenceTypeConversation {
		referenceType = amaicall.ReferenceTypeConversation
	}

	// start service
	sv, err := h.reqHandler.AIV1ServiceTypeAIcallStart(ctx, opt.AIID, af.ID, referenceType, af.ReferenceID, false, amaicall.GenderNone, "", 30000)
	if err != nil {
		return errors.Wrap(err, "Could not start the service.")
	}
	log.WithField("service", sv).Debugf("Started service. service_type: %s, service_id: %s", sv.Type, sv.ID)

	// push the actions
	if errPush := h.PushStack(ctx, af, sv.ID, sv.PushActions); errPush != nil {
		log.Errorf("Could not push the actions to the stack. err: %v", errPush)
		return errors.Wrapf(errPush, "Could not push the actions to the stack.")
	}

	return nil
}
