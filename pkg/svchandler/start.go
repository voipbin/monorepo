package svchandler

import (
	"context"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
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
	// get service
	service := getService(cn)

	switch service {
	case svcEcho:
		return h.serviceEchoStart(cn)

	default:
		// svcNone will get to here.
		// no route found
		h.reqHandler.ChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
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
	c := call.NewCallByChannel(cn, call.TypeEcho, call.DirectionIncoming)
	if err := h.db.CallCreate(context.Background(), c); err != nil {
		return err
	}

	// answer
	if err := h.reqHandler.ChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
		return err
	}

	// set timeout for 180 sec
	if err := h.reqHandler.ChannelVariableSet(c.AsteriskID, c.ChannelID, "TIMEOUT(absolute)", "180"); err != nil {
		return err
	}

	// continue to svc-echo
	if err := h.reqHandler.ChannelContinue(c.AsteriskID, c.ChannelID, "svc-echo", c.Destination.Target, 1, ""); err != nil {
		return err
	}

	return nil
}
