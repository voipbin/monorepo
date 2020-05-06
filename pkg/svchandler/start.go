package svchandler

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"

	log "github.com/sirupsen/logrus"
)

// StasisStart event's context types
const (
	contextIncomingVoip = "in-voipbin"
)

// domain types
const (
	domainEcho = "echo.voipbin.net"
)

type service string

const (
	svcNone service = "none"
	svcEcho service = "echo"
)

// Start starts the call service
func (h *svcHandler) Start(cn *channel.Channel) error {

	if cn.Tech != channel.TechPJSIP {
		// we don't do any other tech at here.
		return nil
	}

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
	case contextIncomingVoip:
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
func (h *svcHandler) serviceEchoStart(cn *channel.Channel) error {

	option := action.OptionEcho{
		Duration: 180,
		DTMF:     true,
	}

	opt, err := json.Marshal(option)
	if err != nil {
		return err
	}

	// create a action echo
	action := &action.Action{
		ID:     uuid.Nil,
		Type:   action.TypeEcho,
		Option: opt,
		Next:   uuid.Nil,
	}

	c := call.NewCallByChannel(cn, call.TypeEcho, call.DirectionIncoming)
	if err := h.db.CallCreate(context.Background(), c); err != nil {
		return err
	}

	// set flowid
	if err := h.db.CallSetFlowID(context.Background(), c.ID, uuid.Nil, getCurTime()); err != nil {
		return err
	}

	// set action
	log.Infof("%v", action)

	// answer
	if err := h.reqHandler.AstChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
		return err
	}

	// set timeout for 180 sec
	if err := h.reqHandler.AstChannelVariableSet(c.AsteriskID, c.ChannelID, "TIMEOUT(absolute)", "180"); err != nil {
		return err
	}

	// continue to svc-echo
	if err := h.reqHandler.AstChannelContinue(c.AsteriskID, c.ChannelID, "svc-echo", c.Destination.Target, 1, ""); err != nil {
		return err
	}

	return nil
}
