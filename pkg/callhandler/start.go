package callhandler

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// StasisStart event's context types
const (
	contextIncomingCall = "call-in"
)

// domain types
const (
	domainEcho       = "echo.voipbin.net"
	domainConference = "conference.voipbin.net"
	domainSipService = "sip-service.voipbin.net"
)

// default max timeout for each services. sec.
const (
	defaultMaxTimeoutEcho       = "300"   // maximum call duration for service echo. 5 min
	defaultMaxTimeoutConference = "10800" // maximum call duration for service conf-soft. 3 hours
)

func (h *callHandler) createCall(ctx context.Context, c *call.Call) error {
	if err := h.db.CallCreate(ctx, c); err != nil {
		return err
	}
	promCallCreateTotal.WithLabelValues(string(c.Direction), string(c.Type)).Inc()

	return nil
}

// Start starts the call service
func (h *callHandler) Start(cn *channel.Channel) error {

	// get service
	cType := getType(cn)

	switch cType {
	case call.TypeEcho:
		return h.typeEchoStart(cn)

	case call.TypeConference:
		return h.typeConferenceStart(cn)

	case call.TypeSipService:
		return h.typeSipServiceStart(cn)

	default:
		// call.TypeNone will get to here.
		// no route found
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("no route found for stasisstart. asterisk_id: %s, channel_id: %s", cn.AsteriskID, cn.ID)
	}
}

// getType returns correct service type
// it checks context first,
// then checks another event arguments for getting service
func getType(cn *channel.Channel) call.Type {
	context := cn.Data["CONTEXT"]

	switch context {
	case contextIncomingCall:
		return getTypeContextIncomingCall(cn)

	default:
		return call.TypeNone
	}
}

// getTypeContextIncomingCall returns the service type for incoming call context
func getTypeContextIncomingCall(cn *channel.Channel) call.Type {
	// all of the incoming calls are hitting the same context.
	// so we have to distinguish them using the requested domain name.
	domain := cn.Data["DOMAIN"]
	switch domain {
	case domainEcho:
		return call.TypeEcho

	case domainConference:
		return call.TypeConference

	case domainSipService:
		return call.TypeSipService

	default:
		return call.TypeNone
	}
}

// stasisStartServiceEcho handles echo calltype request.
func (h *callHandler) typeEchoStart(cn *channel.Channel) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
		})

	// set absolute timeout for 300 sec
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a timeout for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	c := call.NewCallByChannel(cn, call.TypeEcho, call.DirectionIncoming)
	if err := h.createCall(ctx, c); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not create a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call":      c.ID,
			"type":      c.Type,
			"direction": c.Direction,
		})
	log.Debug("The call has created.")

	// set flowid
	if err := h.db.CallSetFlowID(ctx, c.ID, uuid.Nil); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a flow id for call. call: %s, err: %v", c.ID, err)
	}

	// create an option for action echo
	// create default option for echo
	option := action.OptionEcho{
		Duration: 180 * 1000, // duration 180 sec
		DTMF:     true,
	}
	opt, err := json.Marshal(option)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not marshal the option. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create an action
	action := &action.Action{
		ID:     action.IDBegin,
		Type:   action.TypeEcho,
		Option: opt,
		Next:   action.IDEnd,
	}

	c, err = h.db.CallGet(ctx, c.ID)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get created call info. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	// execute action
	return h.ActionExecute(c, action)
}

// serviceConferenceStart handles conference calltype start.
func (h *callHandler) typeConferenceStart(cn *channel.Channel) error {
	ctx := context.Background()
	cfID := uuid.FromStringOrNil(cn.DestinationNumber)

	log := log.WithFields(
		log.Fields{
			"channel":    cn.ID,
			"asterisk":   cn.AsteriskID,
			"conference": cfID,
		})
	log.Debugf("Starting the conference to joining. source: %s", cn.SourceNumber)

	// get conference info
	cf, err := h.db.ConferenceGet(ctx, cfID)
	if err != nil {
		log.Debug("The conference has not created.")
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	// Set absolute timeout for conference
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutConference); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a timeout for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create a call
	c := call.NewCallByChannel(cn, call.TypeConference, call.DirectionIncoming)
	if err := h.createCall(ctx, c); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not create a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call":      c.ID,
			"type":      c.Type,
			"direction": c.Direction,
		})
	log.Debug("The call has created.")

	// set flowid
	if err := h.db.CallSetFlowID(ctx, c.ID, uuid.Nil); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a flow id for call. call: %s, err: %v", c.ID, err)
	}

	// create an option for action echo
	// create default option for echo
	option := action.OptionConferenceJoin{
		ConferenceID: cf.ID.String(),
	}
	opt, err := json.Marshal(option)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not marshal the option. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create an action
	action := &action.Action{
		ID:     action.IDBegin,
		Type:   action.TypeConferenceJoin,
		Option: opt,
		Next:   action.IDEnd,
	}

	c, err = h.db.CallGet(ctx, c.ID)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get created call info. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	// execute action
	return h.ActionExecute(c, action)
}

// typeSipServiceStart handles sip-service calltype request.
func (h *callHandler) typeSipServiceStart(cn *channel.Channel) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
		})

	// set absolute timeout for 300 sec
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a timeout for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create a call
	c := call.NewCallByChannel(cn, call.TypeEcho, call.DirectionIncoming)
	if err := h.createCall(ctx, c); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not create a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call":      c.ID,
			"type":      c.Type,
			"direction": c.Direction,
		})
	log.Debug("Created a call.")

	// set flowid
	// because this call type support only 1 action, we don't set any valid call-flow id here
	if err := h.db.CallSetFlowID(ctx, c.ID, uuid.Nil); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a flow id for call. call: %s, err: %v", c.ID, err)
	}

	var act *action.Action = nil
	switch c.Destination.Target {
	case string(action.TypeEcho):
		// create an option for action echo
		// create default option for echo
		option := action.OptionEcho{
			Duration: 180 * 1000, // duration 180 sec
			DTMF:     true,
		}
		opt, err := json.Marshal(option)
		if err != nil {
			h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
			return fmt.Errorf("Could not marshal the option. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
		}

		// create an action
		act = &action.Action{
			ID:     action.IDBegin,
			Type:   action.TypeEcho,
			Option: opt,
			Next:   action.IDEnd,
		}
	}

	c, err := h.db.CallGet(ctx, c.ID)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get created call info. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	// execute action
	return h.ActionExecute(c, act)
}
