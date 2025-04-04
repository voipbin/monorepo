package callhandler

import (
	"context"
	"fmt"
	"time"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	callapplication "monorepo/bin-call-manager/models/callapplication"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/recording"
)

// Redirect options for timeout action
const (
	redirectTimeoutContext  = "svc-stasis"
	redirectTimeoutExten    = "s"
	redirectTimeoutPriority = "1"
)

// cleanCurrentAction cleans the given call's current blocking action.
// return true if it needs to get next action.
func (h *callHandler) cleanCurrentAction(ctx context.Context, c *call.Call) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "cleanCurrentAction",
		"call_id":           c.ID,
		"current_action_id": c.Action.ID,
	})

	// get channel
	cn, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		return false, err
	}

	// check channel's playback.
	if cn.PlaybackID != "" {
		log.WithField("playback_id", cn.PlaybackID).Debug("The channel has playback id. Stopping now.")
		if errStop := h.channelHandler.PlaybackStop(ctx, c.ChannelID); errStop != nil {
			log.Errorf("Could not stop the playback. err: %v", errStop)
		}
		return false, nil
	}

	// check call's confbridge
	if c.ConfbridgeID != uuid.Nil {
		log.WithField("confbridge_id", c.ConfbridgeID).Debug("The call is in the conference. Leaving from the conference now.")
		if err := h.confbridgeHandler.Kick(ctx, c.ConfbridgeID, c.ID); err != nil {
			log.Errorf("Could not kick the call from the confbridge. err: %v", err)
		}
		return false, nil
	}

	return true, nil
}

// actionExecute execute the action withe the call
func (h *callHandler) actionExecute(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ActionExecute",
		"call_id": c.ID,
		"action":  c.Action,
	})
	log.Debugf("Executing the action. action_type: %s", c.Action.Type)

	start := time.Now()
	var err error

	if c.Action.ID == fmaction.IDEmpty {
		log.Infof("The action_id is empty. Consider failed to execute the first action. Hangup the call. action_id: %s", c.Action.ID)
		c.Action.Type = fmaction.TypeHangup
	} else if c.Action.ID == fmaction.IDFinish {
		log.Infof("The action_id is finish. Hangup the call. action_id: %s", c.Action.ID)
		c.Action.Type = fmaction.TypeHangup
	}

	switch c.Action.Type {
	case fmaction.TypeAMD:
		err = h.actionExecuteAMD(ctx, c)

	case fmaction.TypeAnswer:
		err = h.actionExecuteAnswer(ctx, c)

	case fmaction.TypeBeep:
		err = h.actionExecuteBeep(ctx, c)

	case fmaction.TypeConfbridgeJoin:
		err = h.actionExecuteConfbridgeJoin(ctx, c)

	case fmaction.TypeDigitsReceive:
		err = h.actionExecuteDigitsReceive(ctx, c)

	case fmaction.TypeDigitsSend:
		err = h.actionExecuteDigitsSend(ctx, c)

	case fmaction.TypeEcho:
		err = h.actionExecuteEcho(ctx, c)

	case fmaction.TypeExternalMediaStart:
		err = h.actionExecuteExternalMediaStart(ctx, c)

	case fmaction.TypeExternalMediaStop:
		err = h.actionExecuteExternalMediaStop(ctx, c)

	case fmaction.TypeHangup:
		err = h.actionExecuteHangup(ctx, c)

	case fmaction.TypePlay:
		err = h.actionExecutePlay(ctx, c)

	case fmaction.TypeRecordingStart:
		err = h.actionExecuteRecordingStart(ctx, c)

	case fmaction.TypeRecordingStop:
		err = h.actionExecuteRecordingStop(ctx, c)

	case fmaction.TypeSleep:
		err = h.actionExecuteSleep(ctx, c)

	case fmaction.TypeStreamEcho:
		err = h.actionExecuteStreamEcho(ctx, c)

	case fmaction.TypeTalk:
		err = h.actionExecuteTalk(ctx, c)

	default:
		log.Errorf("Could not find action handle found. call: %s, action: %s, type: %s", c.ID, c.Action.ID, c.Action.Type)
		err = fmt.Errorf("no action handler found")
	}
	elapsed := time.Since(start)
	promCallActionProcessTime.WithLabelValues(string(c.Action.Type)).Observe(float64(elapsed.Milliseconds()))

	//  if the action execution has failed move to the next action
	if err != nil {
		log.Errorf("Could not execute the action correctly. Move to next action. err: %v", err)
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}

	return nil
}

// ActionNext Execute next action
func (h *callHandler) ActionNext(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "ActionNext",
		"call": c,
	})
	log.Debug("Getting a next action for the call.")

	if c.Status == call.StatusHangup {
		log.Debug("The call has hungup already.")
		return nil
	}

	if c.ActiveflowID == uuid.Nil {
		log.Info("No activeflow id found. Hangup the call.")
		_, _ = h.HangingUp(ctx, c.ID, call.HangupReasonNormal)
		return nil
	}

	if c.ActionNextHold {
		log.Debug("The action next is on hold. Nothing to do.")
		return nil
	}

	// set action next hold
	if errHold := h.updateActionNextHold(ctx, c.ID, true); errHold != nil {
		log.Errorf("Could not set the action next hold. err: %v", errHold)
		_, _ = h.HangingUp(ctx, c.ID, call.HangupReasonNormal)
		return nil
	}

	// get next action
	nextAction, err := h.reqHandler.FlowV1ActiveflowGetNextAction(ctx, c.ActiveflowID, c.Action.ID)
	if err != nil {
		// could not get the next action from the flow-manager.
		// we don't hangup the call here. because it is possible to fetching the action with wrong current action id.
		log.WithField("action", c.Action).Infof("Could not get the next action from the flow-manager. err: %v", err)
		return nil
	}
	log.WithField("action", nextAction).Debugf("Received next action. action_id: %s, action_type: %s", nextAction.ID, nextAction.Type)

	// set action and action next hold
	nextAction.TMExecute = h.utilHandler.TimeGetCurTime()
	cc, err := h.updateActionAndActionNextHold(ctx, c.ID, nextAction)
	if err != nil {
		log.Errorf("Could not set the action for call. Move to the next action. err: %v", err)
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}

	if err := h.actionExecute(ctx, cc); err != nil {
		log.Errorf("Could not execute the next action correctly. Hanging up the call. err: %v", err)
		_, _ = h.HangingUp(ctx, c.ID, call.HangupReasonNormal)
		return nil
	}

	return nil
}

// ActionNextForce Execute next action forcedly
func (h *callHandler) ActionNextForce(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "ActionNextForce",
		"call": c,
	})
	log.Debug("Getting a next action for the call.")

	// set action next unhold
	if errHold := h.updateActionNextHold(ctx, c.ID, false); errHold != nil {
		log.Errorf("Could not set the action next hold. err: %v", errHold)
		_, _ = h.HangingUp(ctx, c.ID, call.HangupReasonNormal)
		return nil
	}

	// cleanup the call's current action
	needNext, err := h.cleanCurrentAction(ctx, c)
	if err != nil {
		log.Errorf("Could not cleanup the current action. err: %v", err)
	}

	if !needNext {
		return nil
	}

	return h.ActionNext(ctx, c)
}

// ActionTimeout handles action's timeout
func (h *callHandler) ActionTimeout(ctx context.Context, callID uuid.UUID, a *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ActionTimeout",
		"call_id":     callID,
		"action_id":   a.ID,
		"action_type": a.Type,
	})
	log.WithField("action", a).Infof("The call's action has timed out.")

	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		return err
	}
	log.WithField("call", c).Debugf("Found call info. call_id: %s", c.ID)

	// check current action and requested action info
	if (c.Action.ID != a.ID) || (c.Action.TMExecute != a.TMExecute) || c.ActionNextHold {
		return fmt.Errorf("invalid timed out action condition")
	}

	// get channel
	cn, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. channel_id: %s, err: %v", c.ChannelID, err)
		return err
	}

	// check the channel is in the stasis.
	// if the channel is not in the stasis, send the AMI redirect request.
	switch cn.StasisName {

	// not in the stasis
	// need to be redirected to the redirectTimeoutContext.
	case "":
		return h.channelHandler.Redirect(ctx, c.ChannelID, redirectTimeoutContext, redirectTimeoutExten, redirectTimeoutPriority)

	// in the stasis
	// send a request for the execute next call action
	default:
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}
}

// actionExecuteAnswer executes the action type answer
func (h *callHandler) actionExecuteAnswer(ctx context.Context, c *call.Call) error {

	var option fmaction.OptionAnswer
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	if errAnswer := h.channelHandler.Answer(ctx, c.ChannelID); errAnswer != nil {
		return errors.Wrap(errAnswer, "could not answer")
	}

	// send next action request
	if errAction := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); errAction != nil {
		return errors.Wrapf(errAction, "could not send the next action request. err: %v", errAction)
	}

	return nil
}

// actionExecuteBeep executes the action type beep
func (h *callHandler) actionExecuteBeep(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteBeep",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	var option fmaction.OptionBeep
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	// create a media string array
	medias := []string{
		"sound:beep",
	}
	log.WithFields(logrus.Fields{
		"media": medias,
	}).Debugf("Sending a request to the asterisk for media playing.")

	// play the beep.
	if errPlay := h.channelHandler.Play(ctx, c.ChannelID, c.Action.ID, medias, ""); errPlay != nil {
		log.Errorf("Could not play the media. media: %v, err: %v", medias, errPlay)
		return fmt.Errorf("could not play the media. err: %v", errPlay)
	}

	return nil
}

// actionExecuteEcho executes the action type echo
func (h *callHandler) actionExecuteEcho(ctx context.Context, c *call.Call) error {
	var option fmaction.OptionEcho
	if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
		return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
	}

	// set default duration if it is not set correctly
	if option.Duration <= 0 {
		option.Duration = 180 * 1000 // default duration 180 sec
	}

	// continue the extension
	if errContinue := h.channelHandler.Continue(ctx, c.ChannelID, "svc-echo", "s", 1, ""); errContinue != nil {
		return fmt.Errorf("could not continue the call for action. call: %s, action: %s, err: %v", c.ID, c.Action.ID, errContinue)
	}

	// set timeout
	if errTimeout := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, option.Duration, &c.Action); errTimeout != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, c.Action.ID, errTimeout)
	}

	return nil
}

// actionExecuteConfbridgeJoin executes the action type ConferenceEnter
func (h *callHandler) actionExecuteConfbridgeJoin(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteConfbridgeJoin",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	var option fmaction.OptionConfbridgeJoin
	if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
		return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
	}

	// join to the confbridge
	if err := h.confbridgeHandler.Join(ctx, option.ConfbridgeID, c.ID); err != nil {
		log.Errorf("Could not join to the confbridge. call: %s, err: %v", c.ID, err)
		return err
	}

	return nil
}

// actionExecutePlay executes the action type play
func (h *callHandler) actionExecutePlay(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecutePlay",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	var option fmaction.OptionPlay
	if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
		return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
	}

	if errPlay := h.Play(ctx, c.ID, true, option.StreamURLs); errPlay != nil {
		log.Errorf("Could not play the media correctly. err: %v", errPlay)
		return errors.Wrap(errPlay, "could not play the media correctly")
	}

	return nil
}

// actionExecuteStreamEcho executes the action type stream_echo
// stream_echo does not support timeout and it's blocking action.
// need to set the channel timeout before call this action.
func (h *callHandler) actionExecuteStreamEcho(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteStreamEcho",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})
	log.Debug("Executing action.")

	var option fmaction.OptionStreamEcho
	if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
		return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
	}

	// set default duration if it is not set correctly
	if option.Duration <= 0 {
		option.Duration = 180 * 1000 // default duration 180 sec
	}

	// continue the extension
	if errContinue := h.channelHandler.Continue(ctx, c.ChannelID, "svc-stream_echo", "s", 1, ""); errContinue != nil {
		return fmt.Errorf("could not continue the call for action. call: %s, action: %s, err: %v", c.ID, c.Action.ID, errContinue)
	}

	// set timeout
	// send delayed message for next action execution after 10 ms.
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, option.Duration, &c.Action); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, c.Action.ID, err)
	}

	return nil
}

// actionExecuteHangup executes the action type hangup
func (h *callHandler) actionExecuteHangup(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteHangup",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	var option fmaction.OptionHangup
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	if option.ReferenceID != uuid.Nil {
		log.Debugf("The hangup has reference info. Hanging up the call with reference info. reference_id: %s", option.ReferenceID)
		_, err := h.hangingupWithReference(ctx, c, option.ReferenceID)
		if err != nil {
			log.Errorf("Could not hanging up the call with reference info. Hanging up the call with default reason. err: %v", err)
			_, _ = h.HangingUp(ctx, c.ID, call.HangupReasonNormal)
		}
		return nil
	}

	// hangup has no reference id.
	// get hangup reaon from the option.
	reason := call.HangupReason(option.Reason)
	if reason == call.HangupReasonNone {
		reason = call.HangupReasonNormal
	}

	tmp, err := h.HangingUp(ctx, c.ID, reason)
	if err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
		return nil
	}
	log.WithField("call", tmp).Debugf("Hanging up the call with no reference info. call_id: %s", tmp.ID)

	return nil
}

// actionExecuteTalk executes the action type talk
func (h *callHandler) actionExecuteTalk(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteTalk",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})
	log.WithField("call", c).Debugf("Executing talk.")

	var option fmaction.OptionTalk
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	if errTalk := h.Talk(ctx, c.ID, true, option.Text, option.Gender, option.Language); errTalk != nil {
		log.Errorf("Could not talk correctly. err: %v", errTalk)
		return errors.Wrap(errTalk, "could not talk correctly")
	}

	return nil
}

// actionExecuteRecordingStart executes the action type recording_start.
// It starts recording.
func (h *callHandler) actionExecuteRecordingStart(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteRecordingStart",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	var option fmaction.OptionRecordingStart
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	tmp, err := h.RecordingStart(
		ctx,
		c.ID,
		recording.Format(option.Format),
		option.EndOfSilence,
		option.EndOfKey,
		option.Duration,
		option.OnEndFlowID,
	)
	if err != nil {
		log.Errorf("Could not start the recording. err: %v", err)
		return err
	}
	log.WithField("call", tmp).Debugf("Started recording. call_id: %s", c.ID)

	// send next action request
	if errNext := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); errNext != nil {
		return fmt.Errorf("could not send the next action request. err: %v", errNext)
	}

	return nil
}

// actionExecuteRecordingStop executes the action type recording_stop.
// It stops recording.
func (h *callHandler) actionExecuteRecordingStop(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteRecordingStop",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	var option fmaction.OptionRecordingStop
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	tmp, errRecording := h.RecordingStop(ctx, c.ID)
	if errRecording != nil {
		// we failed to stop the recording. But we still want to continue to call process.
		log.Errorf("Could not stop the recording. recording_id: %s, err: %v", c.RecordingID, errRecording)
	}
	log.WithField("call", tmp).Debugf("Stopped recording. call_id: %s, recording_id: %s", c.ID, c.RecordingID)

	if err := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); err != nil {
		log.Errorf("Could not execute next action call. err: %v", err)
		return err
	}

	return nil
}

// actionExecuteDigitsReceive executes the action type dtmf_receive.
// It collects the dtmfs within duration.
func (h *callHandler) actionExecuteDigitsReceive(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteDigitsReceive",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	var option fmaction.OptionDigitsReceive
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	condition, err := h.checkDigitsCondition(ctx, c.ActiveflowID, &option)
	if err != nil {
		log.Errorf("Could not validate digits condition. err: %v", err)
		return err
	}

	if condition {
		log.Debugf("The stored dtmfs are already qualified the finish condition.")
		if err := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); err != nil {
			return fmt.Errorf("could not send the next action request. err: %v", err)
		}
		return nil
	}

	// set timeout
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, option.Duration, &c.Action); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, c.Action.ID, err)
	}

	return nil
}

// actionExecuteDigitsSend executes the action type dtmf_send.
// It sends the DTMFs to the call.
func (h *callHandler) actionExecuteDigitsSend(ctx context.Context, c *call.Call) error {
	var option fmaction.OptionDigitsSend
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	// send dtmfs
	if errDTMF := h.channelHandler.DTMFSend(ctx, c.ChannelID, option.Digits, option.Duration, 0, option.Interval, 0); errDTMF != nil {
		return errors.Wrap(errDTMF, "could not send the dtmfs")
	}

	// caculate timeout
	// because of the Asterisk doesn't send the dtmf send finish event, we need to do this.
	maxTimeout := (option.Duration * len(option.Digits))
	if len(option.Digits) > 1 {
		maxTimeout = maxTimeout + (option.Interval * (len(option.Digits) - 1))
	}

	// set timeout
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, maxTimeout, &c.Action); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, c.Action.ID, err)
	}

	return nil
}

// actionExecuteExternalMediaStart executes the action type external_media_start.
func (h *callHandler) actionExecuteExternalMediaStart(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteExternalMediaStart",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	var option fmaction.OptionExternalMediaStart
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}

	cc, err := h.ExternalMediaStart(ctx, c.ID, uuid.Nil, option.ExternalHost, externalmedia.Encapsulation(option.Encapsulation), externalmedia.Transport(option.Transport), option.ConnectionType, option.Format, option.Direction)
	if err != nil {
		log.Errorf("Could not start external media. err: %v", err)
		return err
	}
	log.WithField("call", cc).Debugf("Started external media. external_media_id: %s", cc.ExternalMediaID)

	// send next action request
	return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
}

// actionExecuteExternalMediaStop executes the action type external_media_stop.
func (h *callHandler) actionExecuteExternalMediaStop(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "actionExecuteExternalMediaStop",
		"call_id":     c.ID,
		"action_id":   c.Action.ID,
		"action_type": c.Action.Type,
	})

	if c.ExternalMediaID == uuid.Nil {
		// nothing to do here
		log.Infof("The call has no external media. call_id: %s", c.ID)
	} else {
		// stop the external media
		tmp, err := h.externalMediaHandler.Stop(ctx, c.ExternalMediaID)
		if err != nil {
			log.Errorf("Could not stop the external media. err: %v", err)
			return err
		}
		log.WithField("external_media", tmp).Debugf("Stopped external media. external_media_id: %s", tmp.ID)
	}

	// send next action request
	return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
}

// actionExecuteAMD executes the action type external_media_start.
func (h *callHandler) actionExecuteAMD(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "actionExecuteAMD",
		"call_id":   c.ID,
		"action_id": c.Action.ID,
	})

	var option fmaction.OptionAMD
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}
	log.Debugf("Parsed option. option: %v", option)

	// create a snoop channel
	// set app args
	appArgs := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.ContextTypeCall,
		channel.StasisDataTypeContext, channel.ContextApplication,
		channel.StasisDataTypeCallID, c.ID,
		channel.StasisDataTypeApplicationName, applicationAMD,
	)

	snoopID := h.utilHandler.UUIDCreate().String()
	tmp, err := h.channelHandler.StartSnoop(ctx, c.ChannelID, snoopID, appArgs, channel.SnoopDirectionBoth, channel.SnoopDirectionBoth)
	if err != nil {
		log.Errorf("Could not create a snoop channel for the AMD. error: %v", err)
		return errors.Wrap(err, "could not create a snoop channel for the AMD")
	}
	log.WithField("snoop", tmp).Debugf("Created a new snoop channel. channel_id: %s", tmp.ID)

	// create callapplication info
	app := &callapplication.AMD{
		CallID:        c.ID,
		MachineHandle: string(option.MachineHandle),
		Async:         option.Async,
	}

	// add the amd info to the cache
	if errAMD := h.db.CallApplicationAMDSet(ctx, snoopID, app); errAMD != nil {
		log.Errorf("Could not set the callapplication amd option. err: %v", errAMD)
		_, _ = h.channelHandler.HangingUp(ctx, tmp.ID, ari.ChannelCauseNormalClearing)
	}

	if app.Async {
		// send next action request
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}

	return nil
}

// actionExecuteSleep executes the action type sleep.
func (h *callHandler) actionExecuteSleep(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "actionExecuteSleep",
		"call_id":   c.ID,
		"action_id": c.Action.ID,
	})

	var option fmaction.OptionSleep
	if c.Action.Option != nil {
		if errParse := fmaction.ParseOption(c.Action.Option, &option); errParse != nil {
			return errors.Wrapf(errParse, "could not parse the option. action: %v, err: %v", c.Action, errParse)
		}
	}
	log.Debugf("Parsed option. option: %v", option)

	// set timeout
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, option.Duration, &c.Action); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, c.Action.ID, err)
	}

	return nil
}
