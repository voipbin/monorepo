package callhandler

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// StasisStart event's context types
const (
	contextIncomingVoip = "in-voipbin"
	contextIncomingCall = "call-in"
)

// domain types
const (
	domainEcho = "echo.voipbin.net"
)

type service string

const (
	svcNone     service = "none"
	svcEcho     service = "echo"
	svcConfEcho service = "conf-echo"
)

const (
	defaultMaxTimeoutEcho = "300" // maximum call duration for service echo
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
	service := getService(cn)

	switch service {
	case svcEcho:
		return h.serviceEchoStart(cn)

	default:
		// svcNone will get to here.
		// no route found
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("no route found for stasisstart. asterisk_id: %s, channel_id: %s", cn.AsteriskID, cn.ID)
	}
}

// getService returns correct service type
// it checks context first,
// then checks another event arguments for getting service
func getService(cn *channel.Channel) service {
	context := cn.Data["CONTEXT"]

	switch context {
	case contextIncomingCall, contextIncomingVoip:
		// incoming context
		domain := cn.Data["DOMAIN"]
		switch domain {
		case domainEcho:
			return svcEcho
		}
		// no domain found
		return svcNone
	}

	// no suitable context handler found
	return svcNone
}

// stasisStartServiceEcho handles echo domain request.
func (h *callHandler) serviceEchoStart(cn *channel.Channel) error {
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
