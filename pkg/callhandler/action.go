package callhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
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
		"call":        c.ID,
		"action":      a.ID,
		"action_type": a.Type,
	})
	log.Debug("Executing the action.")

	var err error
	switch a.Type {
	case action.TypeAnswer:
		err = h.actionExecuteAnswer(c, a)

	case action.TypeConferenceJoin:
		err = h.actionExecuteConferenceJoin(c, a)

	case action.TypeEcho:
		err = h.actionExecuteEcho(c, a)

	case action.TypeHangup:
		err = h.actionExecuteHangup(c, a)

	case action.TypePlay:
		err = h.actionExecutePlay(c, a)

	case action.TypeStreamEcho:
		err = h.actionExecuteStreamEcho(c, a)

	default:
		log.Errorf("Could not find action handle found. call: %s, action: %s, type: %s", c.ID, a.ID, a.Type)
		err = fmt.Errorf("no action handler found")
	}

	//  if the action execution has failed move to the next action
	if err != nil {
		log.Errorf("Could not execute the action correctly. Move to next action. err: %v", err)
		return h.reqHandler.CallCallActionNext(c.ID)
	}

	return nil
}

// ActionNext Execute next action
func (h *callHandler) ActionNext(c *call.Call) error {
	log := log.WithFields(
		logrus.Fields{
			"call": c.ID,
			"flow": c.FlowID,
		})
	log.WithFields(
		logrus.Fields{
			"action": c.Action,
		},
	).Debug("Getting a next action for the call.")

	// get next action
	nextAction, err := h.reqHandler.FlowActvieFlowNextGet(c.ID, c.Action.ID)
	if err != nil {
		log.Debugf("Could not get next action from the flow-manager. err: %v", err)
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return err
	}

	return h.ActionExecute(c, nextAction)
}

// ActionTimeout handles action's timeout
func (h *callHandler) ActionTimeout(callID uuid.UUID, a *action.Action) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"call":   callID,
			"action": a.ID,
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
	switch cn.Stasis {

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

	// set current time
	act.TMExecute = getCurTime()

	log := log.WithFields(
		log.Fields{
			"call":        c.ID,
			"action":      act.ID,
			"action_type": act.Type,
		})

	var option action.OptionAnswer
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

	if err := h.reqHandler.AstChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
		return fmt.Errorf("could not answer the call. err: %v", err)
	}

	// set timeout
	// send delayed message for next action execution after 10 ms.
	if err := h.reqHandler.CallCallActionTimeout(c.ID, 10, &act); err != nil {
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteEcho executes the action type echo
func (h *callHandler) actionExecuteEcho(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	// set current time
	act.TMExecute = getCurTime()

	log := log.WithFields(
		log.Fields{
			"call":        c.ID,
			"action":      act.ID,
			"action_type": act.Type,
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

// actionExecuteConferenceJoin executes the action type ConferenceJoin
func (h *callHandler) actionExecuteConferenceJoin(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	// set current time
	act.TMExecute = getCurTime()

	log := log.WithFields(
		log.Fields{
			"call":        c.ID,
			"action":      act.ID,
			"action_type": act.Type,
		})

	var option action.OptionConferenceJoin
	if err := json.Unmarshal(act.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse option. action: %v, err: %v", a, err)
	}
	cfID := uuid.FromStringOrNil(option.ConferenceID)

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

	if err := h.confHandler.Join(cfID, c.ID); err != nil {
		log.Errorf("Could not join to the conference. Executing the next action. call: %s, err: %v", c.ID, err)
		return fmt.Errorf("Could not join to the conference. Executing the next action. call: %s, err: %v", c.ID, err)
	}

	return nil
}

// actionExecutePlay executes the action type play
func (h *callHandler) actionExecutePlay(c *call.Call, a *action.Action) error {
	// copy the action
	act := *a

	// set current time
	act.TMExecute = getCurTime()

	log := log.WithFields(
		log.Fields{
			"call":        c.ID,
			"action":      act.ID,
			"action_type": act.Type,
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
	for _, streamURL := range option.StreamURL {
		media := fmt.Sprintf("sound:%s", streamURL)
		medias = append(medias, media)
	}

	// play
	if err := h.reqHandler.AstChannelPlay(c.AsteriskID, c.ChannelID, act.ID, medias); err != nil {
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

	// set current time
	act.TMExecute = getCurTime()

	log := log.WithFields(
		log.Fields{
			"call":        c.ID,
			"action":      act.ID,
			"action_type": act.Type,
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

	// set current time
	act.TMExecute = getCurTime()

	log := log.WithFields(
		log.Fields{
			"call":        c.ID,
			"action":      act.ID,
			"action_type": act.Type,
		})

	var option action.OptionHangup
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

	// hangup
	h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)

	return nil
}
