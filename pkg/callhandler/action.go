package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	callapplication "gitlab.com/voipbin/bin-manager/call-manager.git/models/callapplication"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// Redirect options for timeout action
const (
	redirectTimeoutContext  = "svc-stasis"
	redirectTimeoutExten    = "s"
	redirectTimeoutPriority = "1"
)

// actionExecuteAMD executes the action type external_media_start.
// return true if it needs to get next action.
func (h *callHandler) cleanCurrentAction(ctx context.Context, c *call.Call) (bool, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "cleanCurrentAction",
			"call_id":           c.ID,
			"current_action_id": c.Action.ID,
		},
	)

	// get channel
	cn, err := h.db.ChannelGet(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		return false, err
	}

	// check channel's playback.
	if cn.PlaybackID != "" {
		log.WithField("playback_id", cn.PlaybackID).Debug("The channel has playback id. Stopping now.")
		if err := h.reqHandler.AstPlaybackStop(ctx, cn.AsteriskID, cn.PlaybackID); err != nil {
			log.Errorf("Could not stop the playback. err: %v", err)
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

// setAction sets the action to the call
func (h *callHandler) setAction(c *call.Call, a *fmaction.Action) error {
	// set action
	if err := h.db.CallSetAction(context.Background(), c.ID, a); err != nil {
		return fmt.Errorf("could not set the action for call. call: %s, err: %v", c.ID, err)
	}
	promCallActionTotal.WithLabelValues(string(a.Type)).Inc()

	return nil
}

// ActionExecute execute the action withe the call
func (h *callHandler) ActionExecute(ctx context.Context, c *call.Call, a *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call":   c.ID,
		"action": a,
	})
	log.Debugf("Executing the action. action_type: %s", a.Type)

	start := time.Now()
	var err error

	// set current time
	a.TMExecute = h.util.GetCurTime()

	// set action
	if errSetAction := h.setAction(c, a); errSetAction != nil {
		log.Errorf("Could not set the action for call. Move to the next action. err: %v", errSetAction)
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}

	if a.ID == fmaction.IDFinish {
		log.Infof("The action_id is finish. Hangup the call. action_id: %s", a.ID)
		a.Type = fmaction.TypeHangup
	}

	switch a.Type {
	case fmaction.TypeAMD:
		err = h.actionExecuteAMD(ctx, c, a)

	case fmaction.TypeAnswer:
		err = h.actionExecuteAnswer(ctx, c, a)

	case fmaction.TypeBeep:
		err = h.actionExecuteBeep(ctx, c, a)

	case fmaction.TypeConfbridgeJoin:
		err = h.actionExecuteConfbridgeJoin(ctx, c, a)

	case fmaction.TypeDigitsReceive:
		err = h.actionExecuteDigitsReceive(ctx, c, a)

	case fmaction.TypeDigitsSend:
		err = h.actionExecuteDigitsSend(ctx, c, a)

	case fmaction.TypeEcho:
		err = h.actionExecuteEcho(ctx, c, a)

	case fmaction.TypeExternalMediaStart:
		err = h.actionExecuteExternalMediaStart(ctx, c, a)

	case fmaction.TypeExternalMediaStop:
		err = h.actionExecuteExternalMediaStop(ctx, c, a)

	case fmaction.TypeHangup:
		err = h.actionExecuteHangup(ctx, c, a)

	case fmaction.TypePlay:
		err = h.actionExecutePlay(ctx, c, a)

	case fmaction.TypeRecordingStart:
		err = h.actionExecuteRecordingStart(ctx, c, a)

	case fmaction.TypeRecordingStop:
		err = h.actionExecuteRecordingStop(ctx, c, a)

	case fmaction.TypeSleep:
		err = h.actionExecuteSleep(ctx, c, a)

	case fmaction.TypeStreamEcho:
		err = h.actionExecuteStreamEcho(ctx, c, a)

	case fmaction.TypeTalk:
		err = h.actionExecuteTalk(ctx, c, a)

	default:
		log.Errorf("Could not find action handle found. call: %s, action: %s, type: %s", c.ID, a.ID, a.Type)
		err = fmt.Errorf("no action handler found")
	}
	elapsed := time.Since(start)
	promCallActionProcessTime.WithLabelValues(string(a.Type)).Observe(float64(elapsed.Milliseconds()))

	//  if the action execution has failed move to the next action
	if err != nil {
		log.Errorf("Could not execute the action correctly. Move to next action. err: %v", err)
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}

	return nil
}

// ActionNext Execute next action
func (h *callHandler) ActionNext(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "ActionNext",
			"call_id":           c.ID,
			"flow_id":           c.FlowID,
			"current_action_id": c.Action.ID,
		})
	log.WithFields(
		logrus.Fields{
			"action": c.Action,
		},
	).Debug("Getting a next action for the call.")

	if c.Status == call.StatusHangup {
		log.WithField("call", c).Debug("The call has hungup already.")
		return nil
	}

	if c.FlowID == uuid.Nil {
		log.WithField("call", c).Info("No flow id found. Hangup the call.")
		_ = h.HangingUp(ctx, c.ID, ari.ChannelCauseNormalClearing)
		return nil
	}

	// get next action
	nextAction, err := h.reqHandler.FlowV1ActiveflowGetNextAction(ctx, c.ActiveFlowID, c.Action.ID)
	if err != nil {
		// could not get the next action from the flow-manager.
		log.WithField("action", c.Action).Infof("Could not get the next action from the flow-manager. err: %v", err)
		_ = h.HangingUp(ctx, c.ID, ari.ChannelCauseNormalClearing)
		return nil
	}
	log.Debugf("Received next action. action_id: %s, action_type: %s", nextAction.ID, nextAction.Type)

	if err := h.ActionExecute(ctx, c, nextAction); err != nil {
		log.Errorf("Could not execute the next action correctly. Hanging up the call. err: %v", err)
		_ = h.HangingUp(ctx, c.ID, ari.ChannelCauseNormalClearing)
		return nil
	}

	return nil
}

// ActionNextForce Execute next action forcedly
func (h *callHandler) ActionNextForce(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": c.ID,
			"flow_id": c.FlowID,
			"func":    "ActionNextForce",
		})
	log.WithField("action", c.Action).Debug("Getting a next action for the call.")

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
		"call_id":     callID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "ActionTimeout",
	})

	log.Infof("The call's action has timed out.")

	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		return err
	}

	// check current action and requested action info
	if (c.Action.ID != a.ID) || (c.Action.TMExecute != a.TMExecute) {
		return fmt.Errorf("no not match action")
	}

	// get channel
	cn, err := h.db.ChannelGet(ctx, c.ChannelID)
	if err != nil {
		return err
	}

	// check the channel is in the stasis.
	// if the channel is not in the stasis, send the AMI redirect request.
	switch cn.StasisName {

	// not in the stasis
	// need to be redirected to the redirectTimeoutContext.
	case "":
		return h.reqHandler.AstAMIRedirect(ctx, cn.AsteriskID, cn.ID, redirectTimeoutContext, redirectTimeoutExten, redirectTimeoutPriority)

	// in the stasis
	// send a request for the execute next call action
	default:
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}
}

// actionExecuteAnswer executes the action type answer
func (h *callHandler) actionExecuteAnswer(ctx context.Context, c *call.Call, a *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteAnswer",
	})

	var option fmaction.OptionAnswer
	if a.Option != nil {
		if err := json.Unmarshal(a.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	if err := h.reqHandler.AstChannelAnswer(ctx, c.AsteriskID, c.ChannelID); err != nil {
		return fmt.Errorf("could not answer the call. err: %v", err)
	}

	// send next action request
	if err := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); err != nil {
		return fmt.Errorf("could not send the next action request. err: %v", err)
	}

	return nil
}

// actionExecuteBeep executes the action type beep
func (h *callHandler) actionExecuteBeep(ctx context.Context, c *call.Call, a *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteBeep",
	})

	var option fmaction.OptionBeep
	if a.Option != nil {
		if err := json.Unmarshal(a.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	// create a media string array
	medias := []string{
		"sound:beep",
	}
	log.WithFields(
		logrus.Fields{
			"media": medias,
		},
	).Debugf("Sending a request to the asterisk for media playing.")

	// play the beep
	if err := h.reqHandler.AstChannelPlay(ctx, c.AsteriskID, c.ChannelID, a.ID, medias, ""); err != nil {
		log.Errorf("Could not play the media. media: %v, err: %v", medias, err)
		return fmt.Errorf("could not play the media. err: %v", err)
	}

	return nil
}

// actionExecuteEcho executes the action type echo
func (h *callHandler) actionExecuteEcho(ctx context.Context, c *call.Call, a *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteEcho",
	})

	var option fmaction.OptionEcho
	if err := json.Unmarshal(a.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
	}

	// set default duration if it is not set correctly
	if option.Duration <= 0 {
		option.Duration = 180 * 1000 // default duration 180 sec
	}

	// continue the extension
	if err := h.reqHandler.AstChannelContinue(ctx, c.AsteriskID, c.ChannelID, "svc-echo", "s", 1, ""); err != nil {
		return fmt.Errorf("could not continue the call for action. call: %s, action: %s, err: %v", c.ID, a.ID, err)
	}

	// set timeout
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, option.Duration, a); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, a.ID, err)
	}

	return nil
}

// actionExecuteConfbridgeJoin executes the action type ConferenceEnter
func (h *callHandler) actionExecuteConfbridgeJoin(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteConfbridgeJoin",
	})

	var option fmaction.OptionConfbridgeJoin
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return err
	}

	// join to the confbridge
	if err := h.confbridgeHandler.Join(ctx, option.ConfbridgeID, c.ID); err != nil {
		log.Errorf("Could not join to the confbridge. call: %s, err: %v", c.ID, err)
		return err
	}

	return nil
}

// actionExecutePlay executes the action type play
func (h *callHandler) actionExecutePlay(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecutePlay",
	})

	var option fmaction.OptionPlay
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
	}

	// create a media string array
	var medias []string
	for _, streamURL := range option.StreamURLs {
		media := fmt.Sprintf("sound:%s", streamURL)
		medias = append(medias, media)
	}
	log.WithFields(
		logrus.Fields{
			"media": medias,
		},
	).Debugf("Sending a request to the asterisk for media playing.")

	// play
	if err := h.reqHandler.AstChannelPlay(ctx, c.AsteriskID, c.ChannelID, act.ID, medias, ""); err != nil {
		log.Errorf("Could not play the media. media: %v, err: %v", medias, err)
		return fmt.Errorf("could not play the media. err: %v", err)
	}

	return nil
}

// actionExecuteStreamEcho executes the action type stream_echo
// stream_echo does not support timeout and it's blocking action.
// need to set the channel timeout before call this action.
func (h *callHandler) actionExecuteStreamEcho(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteStreamEcho",
	})
	log.Debug("Executing action.")

	var option fmaction.OptionStreamEcho
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
	}

	// set default duration if it is not set correctly
	if option.Duration <= 0 {
		option.Duration = 180 * 1000 // default duration 180 sec
	}

	// continue the extension
	if err := h.reqHandler.AstChannelContinue(ctx, c.AsteriskID, c.ChannelID, "svc-stream_echo", "s", 1, ""); err != nil {
		return fmt.Errorf("could not continue the call for action. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	// set timeout
	// send delayed message for next action execution after 10 ms.
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, option.Duration, act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteHangup executes the action type hangup
func (h *callHandler) actionExecuteHangup(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteHangup",
	})

	var option fmaction.OptionHangup
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}

	if err := h.HangingUp(ctx, c.ID, ari.ChannelCauseNormalClearing); err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
	}

	return nil
}

// actionExecuteTalk executes the action type talk
func (h *callHandler) actionExecuteTalk(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteTalk",
	})
	log.WithField("call", c).Debugf("Executing talk.")

	var option fmaction.OptionTalk
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}

	// answer the call if not answered
	if c.Status != call.StatusProgressing {
		if errAnswer := h.reqHandler.AstChannelAnswer(ctx, c.AsteriskID, c.ChannelID); errAnswer != nil {
			log.Errorf("Could not answer the call. err: %v", errAnswer)
			return fmt.Errorf("could not answer the call. err: %v", errAnswer)
		}
	}

	// send request for create wav file
	filename, err := h.reqHandler.TTSV1SpeecheCreate(ctx, c.ID, option.Text, option.Gender, option.Language, 10000)
	if err != nil {
		log.Errorf("Could not create speech file. err: %v", err)
		return fmt.Errorf("could not create tts wav. err: %v", err)
	}
	url := fmt.Sprintf("http://localhost:8000/%s", filename)

	// create a media string array
	var medias []string
	media := fmt.Sprintf("sound:%s", url)
	medias = append(medias, media)

	// play
	if err := h.reqHandler.AstChannelPlay(ctx, c.AsteriskID, c.ChannelID, act.ID, medias, ""); err != nil {
		log.Errorf("Could not play the media for tts. media: %v, err: %v", medias, err)
		return fmt.Errorf("could not play the media for tts. err: %v", err)
	}

	return nil
}

// actionExecuteRecordingStart executes the action type recording_start.
// It starts recording.
func (h *callHandler) actionExecuteRecordingStart(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteRecordingStart",
	})

	var option fmaction.OptionRecordingStart
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}

	// set record format
	format := "wav"
	if option.Format != "" {
		format = option.Format
	}

	recordingID := uuid.Must(uuid.NewV4())
	recordingName := fmt.Sprintf("call_%s_%s", c.ID, getCurTimeRFC3339())
	filename := fmt.Sprintf("%s.%s", recordingName, format)
	channelID := uuid.Must(uuid.NewV4()).String()

	// create a recording
	rec := &recording.Recording{
		ID:          recordingID,
		CustomerID:  c.CustomerID,
		Type:        recording.TypeCall,
		ReferenceID: c.ID,
		Status:      recording.StatusInitiating,
		Format:      format,
		Filename:    filename,

		AsteriskID: c.AsteriskID,
		ChannelID:  channelID,

		TMStart: defaultTimeStamp,
		TMEnd:   defaultTimeStamp,

		TMCreate: h.util.GetCurTime(),
		TMUpdate: defaultTimeStamp,
		TMDelete: defaultTimeStamp,
	}

	if err := h.db.RecordingCreate(ctx, rec); err != nil {
		log.Errorf("Could not create the record. err: %v", err)
		return fmt.Errorf("could not create the record. err: %v", err)
	}

	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s,recording_id=%s,recording_name=%s,format=%s,end_of_silence=%d,end_of_key=%s,duration=%d",
		ContextRecording,
		c.ID,
		recordingID,
		recordingName,
		format,
		option.EndOfSilence,
		option.EndOfKey,
		option.Duration,
	)

	// create a snoop channel
	tmp, err := h.reqHandler.AstChannelCreateSnoop(ctx, rec.AsteriskID, c.ChannelID, rec.ChannelID, appArgs, channel.SnoopDirectionBoth, channel.SnoopDirectionNone)
	if err != nil {
		log.Errorf("Could not create a snoop channel for recroding. err: %v", err)
		return fmt.Errorf("could not create snoop chanel for recrod. err: %v", err)
	}
	log.WithField("channel", tmp).Debugf("Created a new snoop channel. channel_id: %s", tmp.ID)

	// set record channel id
	if err := h.db.CallSetRecordID(ctx, c.ID, recordingID); err != nil {
		log.Errorf("Could not set the record id to the call. err: %v", err)
		return fmt.Errorf("could not set the record id to the call. err: %v", err)
	}

	// add the recording
	if err := h.db.CallAddRecordIDs(ctx, c.ID, recordingID); err != nil {
		log.Errorf("Could not add the record id to the call. err: %v", err)
		return fmt.Errorf("could not add the record id to the call. err: %v", err)
	}

	// send next action request
	if err := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); err != nil {
		log.Errorf("Could not execute next call action. err: %v", err)
		return err
	}

	return nil
}

// actionExecuteRecordingStop executes the action type recording_stop.
// It stops recording.
func (h *callHandler) actionExecuteRecordingStop(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteRecordingStop",
	})

	var option fmaction.OptionRecordingStop
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}

	// we don't do set empty call's recordid at here.
	// setting the recordid will be done with RecordingFinished event.

	// get record
	record, err := h.db.RecordingGet(ctx, c.RecordingID)
	if err != nil {
		log.Errorf("Could not get record info. But keep continue to next. err: %v", err)
	} else {
		// hangup the channel
		if err := h.reqHandler.AstChannelHangup(ctx, record.AsteriskID, record.ChannelID, ari.ChannelCauseNormalClearing, 0); err != nil {
			log.Errorf("Could not hangup the recording channel. err: %v", err)
		}
	}

	// send next action request
	if err := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); err != nil {
		log.Errorf("Could not execute next action call. err: %v", err)
		return err
	}

	return nil
}

// actionExecuteDigitsReceive executes the action type dtmf_receive.
// It collects the dtmfs within duration.
func (h *callHandler) actionExecuteDigitsReceive(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteDigitsReceive",
	})

	var option fmaction.OptionDigitsReceive
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}

	condition, err := h.checkDigitsCondition(ctx, c.ActiveFlowID, &option)
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
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, option.Duration, act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteDigitsSend executes the action type dtmf_send.
// It sends the DTMFs to the call.
func (h *callHandler) actionExecuteDigitsSend(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteDigitsSend",
	})

	var option fmaction.OptionDigitsSend
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}

	// send dtmfs
	if err := h.reqHandler.AstChannelDTMF(ctx, c.AsteriskID, c.ChannelID, option.Digits, option.Duration, 0, option.Interval, 0); err != nil {
		return fmt.Errorf("could not send the dtmfs. err: %v", err)
	}

	// caculate timeout
	// because of the Asterisk doesn't send the dtmf send finish event, we need to do this.
	maxTimeout := (option.Duration * len(option.Digits))
	if len(option.Digits) > 1 {
		maxTimeout = maxTimeout + (option.Interval * (len(option.Digits) - 1))
	}

	// set timeout
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, maxTimeout, act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteExternalMediaStart executes the action type external_media_start.
func (h *callHandler) actionExecuteExternalMediaStart(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteExternalMediaStart",
	})

	// check already external media is going on
	extMedia, _ := h.db.ExternalMediaGet(ctx, c.ID)
	if extMedia != nil {
		log.Infof("The external media is already going on. external_media: %v", extMedia)
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}

	var option fmaction.OptionExternalMediaStart
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}

	extCh, err := h.ExternalMediaStart(ctx, c.ID, true, option.ExternalHost, option.Encapsulation, option.Transport, option.ConnectionType, option.Format, option.Direction)
	if err != nil {
		log.Errorf("Could not start external media. err: %v", err)
		return err
	}
	log.Debugf("Created external media channel. channel: %v", extCh)

	// send next action request
	return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
}

// actionExecuteExternalMediaStop executes the action type external_media_stop.
func (h *callHandler) actionExecuteExternalMediaStop(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   act.ID,
		"action_type": act.Type,
		"func":        "actionExecuteExternalMediaStop",
	})

	// stop the external media
	if err := h.ExternalMediaStop(context.Background(), c.ID); err != nil {
		log.Errorf("Could not stop the external media. err: %v", err)
		return err
	}
	log.Debugf("Stopped external media channel. call_id: %v", c.ID)

	// send next action request
	return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
}

// actionExecuteAMD executes the action type external_media_start.
func (h *callHandler) actionExecuteAMD(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":   c.ID,
		"action_id": act.ID,
		"func":      "actionExecuteAMD",
	})

	var option fmaction.OptionAMD
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}
	log.Debugf("Parsed option. option: %v", option)

	// create a snoop channel
	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s,application_name=%s",
		ContextApplication,
		c.ID,
		applicationAMD,
	)

	snoopID := uuid.Must(uuid.NewV4())
	tmp, err := h.reqHandler.AstChannelCreateSnoop(ctx, c.AsteriskID, c.ChannelID, snoopID.String(), appArgs, channel.SnoopDirectionBoth, channel.SnoopDirectionBoth)
	if err != nil {
		log.Errorf("Could not create a snoop channel for the AMD. error: %v", err)
		return err
	}
	log.WithField("channel", tmp).Debugf("Created a new snoop channel. channel_id: %s", tmp.ID)

	// create callapplication info
	app := &callapplication.AMD{
		CallID:        c.ID,
		MachineHandle: string(option.MachineHandle),
		Async:         option.Async,
	}

	// add the amd info to the cache
	if errAMD := h.db.CallApplicationAMDSet(context.Background(), snoopID.String(), app); errAMD != nil {
		log.Errorf("Could not set the callapplication amd option. err: %v", errAMD)
		_ = h.reqHandler.AstChannelHangup(ctx, c.AsteriskID, snoopID.String(), ari.ChannelCauseNormalClearing, 0)
	}

	if app.Async {
		// send next action request
		return h.reqHandler.CallV1CallActionNext(ctx, c.ID, false)
	}

	return nil
}

// actionExecuteSleep executes the action type sleep.
func (h *callHandler) actionExecuteSleep(ctx context.Context, c *call.Call, act *fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call_id":   c.ID,
		"action_id": act.ID,
		"func":      "actionExecuteSleep",
	})

	var option fmaction.OptionSleep
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", act, err)
		}
	}
	log.Debugf("Parsed option. option: %v", option)

	// set timeout
	if err := h.reqHandler.CallV1CallActionTimeout(ctx, c.ID, option.Duration, act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}
