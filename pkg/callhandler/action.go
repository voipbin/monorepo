package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

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

// setAction sets the action to the call
func (h *callHandler) setAction(c *call.Call, a *action.Action) error {
	// set action
	if err := h.db.CallSetAction(context.Background(), c.ID, a); err != nil {
		return fmt.Errorf("could not set the action for call. call: %s, err: %v", c.ID, err)
	}
	promCallActionTotal.WithLabelValues(string(a.Type)).Inc()

	return nil
}

// ActionExecute execute the action withe the call
func (h *callHandler) ActionExecute(c *call.Call, a *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call":   c.ID,
		"action": a,
	})
	log.Debugf("Executing the action. action_type: %s", a.Type)

	start := time.Now()
	var err error

	// set current time
	a.TMExecute = getCurTime()

	switch a.Type {
	case action.TypeAMD:
		err = h.actionExecuteAMD(c, a)

	case action.TypeAnswer:
		err = h.actionExecuteAnswer(c, a)

	case action.TypeConfbridgeJoin:
		err = h.actionExecuteConfbridgeJoin(c, a)

	case action.TypeDTMFReceive:
		err = h.actionExecuteDTMFReceive(c, a)

	case action.TypeDTMFSend:
		err = h.actionExecuteDTMFSend(c, a)

	case action.TypeEcho:
		err = h.actionExecuteEcho(c, a)

	case action.TypeExternalMediaStart:
		err = h.actionExecuteExternalMediaStart(c, a)

	case action.TypeExternalMediaStop:
		err = h.actionExecuteExternalMediaStop(c, a)

	case action.TypeHangup:
		err = h.actionExecuteHangup(c, a)

	case action.TypePlay:
		err = h.actionExecutePlay(c, a)

	case action.TypeRecordingStart:
		err = h.actionExecuteRecordingStart(c, a)

	case action.TypeRecordingStop:
		err = h.actionExecuteRecordingStop(c, a)

	case action.TypeStreamEcho:
		err = h.actionExecuteStreamEcho(c, a)

	case action.TypeTalk:
		err = h.actionExecuteTalk(c, a)

	default:
		log.Errorf("Could not find action handle found. call: %s, action: %s, type: %s", c.ID, a.ID, a.Type)
		err = fmt.Errorf("no action handler found")
	}
	elapsed := time.Since(start)
	promCallActionProcessTime.WithLabelValues(string(a.Type)).Observe(float64(elapsed.Milliseconds()))

	//  if the action execution has failed move to the next action
	if err != nil {
		log.Errorf("Could not execute the action correctly. Move to next action. err: %v", err)
		return h.reqHandler.CallCallActionNext(c.ID)
	}

	return nil
}

// ActionNext Execute next action
func (h *callHandler) ActionNext(c *call.Call) error {
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": c.ID,
			"flow_id": c.FlowID,
			"func":    "ActionNext",
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
		_ = h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return nil
	}

	// get next action
	nextAction, err := h.reqHandler.FlowActvieFlowNextGet(c.ID, c.Action.ID)
	if err != nil {
		log.Errorf("Could not get the next action from the flow-manager. Hanging up the call. err: %v", err)
		_ = h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return nil
	}
	log.Debugf("Received next action. action_id: %s, action_type: %s", nextAction.ID, nextAction.Type)

	if err := h.ActionExecute(c, nextAction); err != nil {
		log.Errorf("Could not execute the next action correctly. Hanging up the call. err: %v", err)
		_ = h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return nil
	}

	return nil
}

// ActionTimeout handles action's timeout
func (h *callHandler) ActionTimeout(callID uuid.UUID, a *action.Action) error {
	ctx := context.Background()

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
		return h.reqHandler.AstAMIRedirect(cn.AsteriskID, cn.ID, redirectTimeoutContext, redirectTimeoutExten, redirectTimeoutPriority)

	// in the stasis
	// send a request for the execute next call action
	default:
		return h.reqHandler.CallCallActionNext(c.ID)
	}
}

// actionExecuteAnswer executes the action type answer
func (h *callHandler) actionExecuteAnswer(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteAnswer",
	})

	var option action.OptionAnswer
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	// set option
	rawOption, err := json.Marshal(option)
	if err != nil {
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	if err := h.reqHandler.AstChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
		return fmt.Errorf("could not answer the call. err: %v", err)
	}

	// send next action request
	if err := h.reqHandler.CallCallActionNext(c.ID); err != nil {
		return fmt.Errorf("Could not send the next action request. err: %v", err)
	}

	return nil
}

// actionExecuteEcho executes the action type echo
func (h *callHandler) actionExecuteEcho(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteEcho",
	})

	var option action.OptionEcho
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
	}

	// set default duration if it is not set correctly
	if option.Duration <= 0 {
		option.Duration = 180 * 1000 // default duration 180 sec
	}

	// set option
	rawOption, err := json.Marshal(option)
	if err != nil {
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// continue the extension
	if err := h.reqHandler.AstChannelContinue(c.AsteriskID, c.ChannelID, "svc-echo", "s", 1, ""); err != nil {
		return fmt.Errorf("could not continue the call for action. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	// set timeout
	// send delayed message for next action execution after 10 ms.
	if err := h.reqHandler.CallCallActionTimeout(c.ID, option.Duration, &act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteConfbridgeJoin executes the action type ConferenceEnter
func (h *callHandler) actionExecuteConfbridgeJoin(c *call.Call, a *action.Action) error {
	ctx := context.Background()

	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteConfbridgeJoin",
	})

	var option action.OptionConfbridgeJoin
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return err
	}
	cbID := uuid.FromStringOrNil(option.ConfbridgeID)

	// set option
	rawOption, err := json.Marshal(option)
	if err != nil {
		log.Errorf("Could not marshal the action option. err: %v", err)
		return err
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		log.Errorf("Could not set the action for call. err: %v", err)
		return err
	}

	// join to the confbridge
	if err := h.confbridgeHandler.Join(ctx, cbID, c.ID); err != nil {
		log.Errorf("Could not join to the confbridge. call: %s, err: %v", c.ID, err)
		return err
	}

	return nil
}

// actionExecutePlay executes the action type play
func (h *callHandler) actionExecutePlay(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecutePlay",
	})

	var option action.OptionPlay
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
	}

	// set option
	rawOption, err := json.Marshal(option)
	if err != nil {
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
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
	if err := h.reqHandler.AstChannelPlay(c.AsteriskID, c.ChannelID, act.ID, medias, ""); err != nil {
		log.Errorf("Could not play the media. media: %v, err: %v", medias, err)
		return fmt.Errorf("could not play the media. err: %v", err)
	}

	return nil
}

// actionExecuteStreamEcho executes the action type stream_echo
// stream_echo does not support timeout and it's blocking action.
// need to set the channel timeout before call this action.
func (h *callHandler) actionExecuteStreamEcho(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteStreamEcho",
	})
	log.Debug("Executing action.")

	var option action.OptionStreamEcho
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
	}

	// set default duration if it is not set correctly
	if option.Duration <= 0 {
		option.Duration = 180 * 1000 // default duration 180 sec
	}

	// set option
	rawOption, err := json.Marshal(option)
	if err != nil {
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// continue the extension
	if err := h.reqHandler.AstChannelContinue(c.AsteriskID, c.ChannelID, "svc-stream_echo", "s", 1, ""); err != nil {
		return fmt.Errorf("could not continue the call for action. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	// set timeout
	// send delayed message for next action execution after 10 ms.
	if err := h.reqHandler.CallCallActionTimeout(c.ID, option.Duration, &act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteHangup executes the action type hangup
func (h *callHandler) actionExecuteHangup(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteHangup",
	})

	var option action.OptionHangup
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	// set option
	rawOption, err := json.Marshal(option)
	if err != nil {
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	_ = h.HangingUp(c, ari.ChannelCauseNormalClearing)

	return nil
}

// actionExecuteTalk executes the action type talk
func (h *callHandler) actionExecuteTalk(c *call.Call, a *action.Action) error {

	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteTalk",
	})

	var option action.OptionTalk
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// send request for create wav file
	filename, err := h.reqHandler.TTSSpeechesPOST(option.Text, option.Gender, option.Language)
	if err != nil {
		return fmt.Errorf("could not create tts wav. err: %v", err)
	}
	url := fmt.Sprintf("http://localhost:8000/%s", filename)

	// create a media string array
	var medias []string
	media := fmt.Sprintf("sound:%s", url)
	medias = append(medias, media)

	// play
	if err := h.reqHandler.AstChannelPlay(c.AsteriskID, c.ChannelID, act.ID, medias, ""); err != nil {
		log.Errorf("Could not play the media for tts. media: %v, err: %v", medias, err)
		return fmt.Errorf("could not play the media for tts. err: %v", err)
	}

	return nil
}

// actionExecuteRecordingStart executes the action type recording_start.
// It starts recording.
func (h *callHandler) actionExecuteRecordingStart(c *call.Call, a *action.Action) error {

	ctx := context.Background()

	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteRecordingStart",
	})

	var option action.OptionRecordingStart
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	// set record format
	format := "wav"
	if option.Format != "" {
		format = option.Format
	}

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	recordingID := uuid.Must(uuid.NewV4())
	recordingName := fmt.Sprintf("call_%s_%s", c.ID, getCurTimeRFC3339())
	filename := fmt.Sprintf("%s.%s", recordingName, format)
	channelID := uuid.Must(uuid.NewV4()).String()

	// create a recording
	rec := &recording.Recording{
		ID:          recordingID,
		UserID:      c.UserID,
		Type:        recording.TypeCall,
		ReferenceID: c.ID,
		Status:      recording.StatusInitiating,
		Format:      format,
		Filename:    filename,
		WebhookURI:  c.WebhookURI,

		AsteriskID: c.AsteriskID,
		ChannelID:  channelID,

		TMStart: defaultTimeStamp,
		TMEnd:   defaultTimeStamp,

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
		option.Format,
		option.EndOfSilence,
		option.EndOfKey,
		option.Duration,
	)

	// create a snoop channel
	if err := h.reqHandler.AstChannelCreateSnoop(rec.AsteriskID, c.ChannelID, rec.ChannelID, appArgs, channel.SnoopDirectionBoth, channel.SnoopDirectionNone); err != nil {
		log.Errorf("Could not create a snoop channel for recroding. err: %v", err)
		return fmt.Errorf("could not create snoop chanel for recrod. err: %v", err)
	}

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
	if err := h.reqHandler.CallCallActionNext(c.ID); err != nil {
		log.Errorf("Could not execute next call action. err: %v", err)
		return err
	}

	return nil
}

// actionExecuteRecordingStop executes the action type recording_stop.
// It stops recording.
func (h *callHandler) actionExecuteRecordingStop(c *call.Call, a *action.Action) error {
	ctx := context.Background()

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteRecordingStop",
	})

	// copy the action
	act := *a

	var option action.OptionRecordingStop
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// we don't do set empty call's recordid at here.
	// setting the recordid will be done with RecordingFinished event.

	// get record
	record, err := h.db.RecordingGet(ctx, c.RecordingID)
	if err != nil {
		log.Errorf("Could not get record info. But keep continue to next. err: %v", err)
	} else {
		// hangup the channel
		if err := h.reqHandler.AstChannelHangup(record.AsteriskID, record.ChannelID, ari.ChannelCauseNormalClearing); err != nil {
			log.Errorf("Could not hangup the recording channel. err: %v", err)
		}
	}

	// send next action request
	if err := h.reqHandler.CallCallActionNext(c.ID); err != nil {
		log.Errorf("Could not execute next action call. err: %v", err)
		return err
	}

	return nil
}

// actionExecuteDTMFReceive executes the action type dtmf_receive.
// It collects the dtmfs within duration.
func (h *callHandler) actionExecuteDTMFReceive(c *call.Call, a *action.Action) error {
	ctx := context.Background()

	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteDTMFReceive",
	})

	var option action.OptionDTMFReceive
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// get stored dtmf
	dtmfs, err := h.db.CallDTMFGet(ctx, c.ID)

	// check the dtmf finish condition
	if err == nil && len(dtmfs) > 0 && (strings.Contains(option.FinishOnKey, dtmfs) || len(dtmfs) >= option.MaxNumKey) {
		// the stored dtmf has already qualified finish condition.
		log.Debugf("The stored dtmfs are already qualified the finish condition. dtmfs: %s", dtmfs)
		if err := h.reqHandler.CallCallActionNext(c.ID); err != nil {
			return fmt.Errorf("Could not send the next action request. err: %v", err)
		}
		return nil
	}

	// set timeout
	if err := h.reqHandler.CallCallActionTimeout(c.ID, option.Duration, &act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteDTMFSend executes the action type dtmf_send.
// It sends the DTMFs to the call.
func (h *callHandler) actionExecuteDTMFSend(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteDTMFSend",
	})

	var option action.OptionDTMFSend
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// send dtmfs
	if err := h.reqHandler.AstChannelDTMF(c.AsteriskID, c.ChannelID, option.DTMFs, option.Duration, 0, option.Interval, 0); err != nil {
		return fmt.Errorf("Could not send the dtmfs. err: %v", err)
	}

	// caculate timeout
	// because of the Asterisk doesn't send the dtmf send finish event, we need to do this.
	maxTimeout := (option.Duration * len(option.DTMFs))
	if len(option.DTMFs) > 1 {
		maxTimeout = maxTimeout + (option.Interval * (len(option.DTMFs) - 1))
	}

	// set timeout
	if err := h.reqHandler.CallCallActionTimeout(c.ID, maxTimeout, &act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteExternalMediaStart executes the action type external_media_start.
func (h *callHandler) actionExecuteExternalMediaStart(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteExternalMediaStart",
	})

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// check already external media is going on
	ctx := context.Background()
	extMedia, _ := h.db.ExternalMediaGet(ctx, c.ID)
	if extMedia != nil {
		log.Infof("The external media is already going on. external_media: %v", extMedia)
		return h.reqHandler.CallCallActionNext(c.ID)
	}

	var option action.OptionExternalMediaStart
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
		}
	}

	extCh, err := h.ExternalMediaStart(c.ID, true, option.ExternalHost, option.Encapsulation, option.Transport, option.ConnectionType, option.Format, option.Direction)
	if err != nil {
		log.Errorf("Could not start external media. err: %v", err)
		return err
	}
	log.Debugf("Created external media channel. channel: %v", extCh)

	// send next action request
	return h.reqHandler.CallCallActionNext(c.ID)
}

// actionExecuteExternalMediaStop executes the action type external_media_stop.
func (h *callHandler) actionExecuteExternalMediaStop(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":     c.ID,
		"action_id":   a.ID,
		"action_type": a.Type,
		"func":        "actionExecuteExternalMediaStop",
	})

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// stop the external media
	if err := h.ExternalMediaStop(c.ID); err != nil {
		log.Errorf("Could not stop the external media. err: %v", err)
		return err
	}
	log.Debugf("Stopped external media channel. call_id: %v", c.ID)

	// send next action request
	return h.reqHandler.CallCallActionNext(c.ID)
}

// actionExecuteAMD executes the action type external_media_start.
func (h *callHandler) actionExecuteAMD(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	log := logrus.WithFields(logrus.Fields{
		"call_id":   c.ID,
		"action_id": a.ID,
		"func":      "actionExecuteAMD",
	})

	// set action
	if err := h.setAction(c, &act); err != nil {
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	var option action.OptionAMD
	if act.Option != nil {
		if err := json.Unmarshal(act.Option, &option); err != nil {
			log.Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse the option. action: %v, err: %v", a, err)
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
	if errSnoop := h.reqHandler.AstChannelCreateSnoop(c.AsteriskID, c.ChannelID, snoopID.String(), appArgs, channel.SnoopDirectionBoth, channel.SnoopDirectionBoth); errSnoop != nil {
		log.Errorf("Could not create a snoop channel for the AMD. error: %v", errSnoop)
		return errSnoop
	}

	// create callapplication info
	app := &callapplication.AMD{
		CallID:        c.ID,
		MachineHandle: option.MachineHandle,
		Async:         option.Async,
	}

	// add the amd info to the cache
	if errAMD := h.db.CallApplicationAMDSet(context.Background(), snoopID.String(), app); errAMD != nil {
		log.Errorf("Could not set the callapplication amd option. err: %v", errAMD)
		_ = h.reqHandler.AstChannelHangup(c.AsteriskID, snoopID.String(), ari.ChannelCauseNormalClearing)
	}

	if app.Async {
		// send next action request
		return h.reqHandler.CallCallActionNext(c.ID)
	}

	return nil
}
