package callhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
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
	log.Debugf("Executing the action. call: %s, action: %s", c.ID, a.Type)

	switch a.Type {
	case action.TypeEcho:
		return h.actionExecuteEcho(c, a)

	case action.TypeEchoLegacy:
		return h.actionExecuteEchoLegacy(c, a)

	case action.TypeStreamEcho:
		return h.actionExecuteStreamEcho(c, a)

	case action.TypeConferenceJoin:
		return h.actionExecuteConferenceJoin(c, a)

	default:
		return fmt.Errorf("no action handle found. type: %s", a.Type)
	}
}

// ActionNext Execute next action
func (h *callHandler) ActionNext(c *call.Call) error {
	ctx := context.Background()

	log := log.WithFields(
		logrus.Fields{
			"call": c.ID,
		})

	// validate current state
	switch {
	case c.Action.Next == action.IDEnd:
		// last action
		log.Debug("End of call flow. No more next action left.")
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return nil

	case c.Action.ID == c.Action.Next:
		// loop detected
		log.WithFields(
			logrus.Fields{
				"action_current": c.Action.ID.String(),
				"action_next":    c.Action.Next.String(),
			}).Warning("Loop detected. Current and the next action id is the same.")
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return nil

	case c.FlowID == uuid.Nil:
		// invalid flow id
		log.Info("The call's flow id is not valid. Hanging up the call.")
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return nil
	}

	// get next action from flow-manager
	action, err := h.reqHandler.FlowActionGet(c.FlowID, c.Action.Next)
	if err != nil {
		log.Errorf("Could not get next flow action. err: %v", err)
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return err
	}

	return h.ActionExecute(c, action)
}

// ActionTimeout handles action's timeout
func (h *callHandler) ActionTimeout(callID uuid.UUID, a *action.Action) error {
	ctx := context.Background()

	log := log.WithFields(
		logrus.Fields{
			"call":   callID,
			"action": a.ID,
		})

	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		return err
	}

	// check current action and requested action info
	if (c.Action.ID != a.ID) || (c.Action.TMExecute != a.TMExecute) {
		return fmt.Errorf("no not match action")
	}

	// execute the ActionNext with goroutine here
	// and because the action timeout process is already done, return the nil right next.
	go func() {
		if err := h.ActionNext(c); err != nil {
			log.Errorf("Could not execute the next action. err: %v", err)
		}
	}()

	return nil
}

// actionExecuteEcho executes the action type echo
func (h *callHandler) actionExecuteEchoLegacy(c *call.Call, a *action.Action) error {
	ctx := context.Background()

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
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// start echo conference
	reqConf := &conference.Conference{
		Type:    conference.TypeEcho,
		Name:    "echo",
		Detail:  "action echo",
		Timeout: 180, // 3 minutes
	}
	conf, err := h.confHandler.Start(reqConf, c)
	if err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not start a conference for call. call: %s, conference_type: %s, err: %v", c.ID, conference.TypeEcho, err)
	}
	log.Debugf("Conference started. conf: %v", conf)

	// set timeout
	// send delayed message for ti
	if err := h.reqHandler.CallCallActionTimeout(c.ID, option.Duration, &act); err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set action timeout for call. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteEcho executes the action type echo
func (h *callHandler) actionExecuteEcho(c *call.Call, a *action.Action) error {
	ctx := context.Background()

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
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// continue the extension
	if err := h.reqHandler.AstChannelContinue(c.AsteriskID, c.ChannelID, "svc-echo", "s", 1, ""); err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not continue the call for action. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}

// actionExecuteConferenceJoin executes the action type ConferenceJoin
func (h *callHandler) actionExecuteConferenceJoin(c *call.Call, a *action.Action) error {
	ctx := context.Background()

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
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	if err := h.confHandler.Join(cfID, c.ID); err != nil {
		log.Errorf("Could not join to the conference. Executing the next action. call: %s, err: %v", c.ID, err)
		h.ActionNext(c)
	}

	return nil
}

// actionExecuteStreamEcho executes the action type stream_echo
// stream_echo does not support timeout and it's blocking action.
// need to set the channel timeout before call this action.
func (h *callHandler) actionExecuteStreamEcho(c *call.Call, a *action.Action) error {
	ctx := context.Background()

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

	// set option
	rawOption, err := json.Marshal(option)
	if err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not marshal the action option. err: %v", err)
	}
	act.Option = rawOption

	// set action
	if err := h.setAction(c, &act); err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set the action for call. err: %v", err)
	}

	// continue the extension
	if err := h.reqHandler.AstChannelContinue(c.AsteriskID, c.ChannelID, "svc-stream_echo", "s", 1, ""); err != nil {
		h.db.CallSetStatus(ctx, c.ID, call.StatusTerminating, getCurTime())
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not continue the call for action. call: %s, action: %s, err: %v", c.ID, act.ID, err)
	}

	return nil
}
