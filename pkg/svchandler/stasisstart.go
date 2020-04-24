package svchandler

import (
	"context"
	"fmt"

	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
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

// StasisStart serves StasisStart event
func (h *svcHandler) StasisStart(e *ari.StasisStart) error {
	// get service
	service := getService(e)

	switch service {
	case svcEcho:
		return h.stasisStartServiceEcho(e)

	default:
		// svcNone will get to here.
		// no route found
		h.reqHandler.ChannelHangup(e.AsteriskID, e.Channel.ID, ari.HangupNoRouteDestination)
		return fmt.Errorf("no route found for stasisstart. asterisk_id: %s, channel_id: %s", e.AsteriskID, e.Channel.ID)
	}
}

// getService returns correct service type
// it checks context first,
// then checks another event arguments for getting service
func getService(e *ari.StasisStart) service {
	context := e.Args["CONTEXT"]

	switch context {
	case contextIncomingVoip:
		// incoming context
		domain := e.Args["DOMAIN"]
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
func (h *svcHandler) stasisStartServiceEcho(e *ari.StasisStart) error {
	// create a call
	source := call.ParseAddressByCallerID(&e.Channel.Caller)
	destination := call.ParseAddressByDialplan(&e.Channel.Dialplan)
	status := call.ParseStatusByChannelState(e.Channel.State)
	data := map[string]string{}

	c := call.NewCall(
		uuid.NewV4(),
		e.AsteriskID,
		e.Channel.ID,
		uuid.Nil,
		call.TypeEcho,

		*source,
		*destination,

		status,
		data,
		call.DirectionIncoming,

		e.Timestamp,
	)

	if err := h.db.CallCreate(context.Background(), c); err != nil {
		return err
	}

	// answer
	if err := h.reqHandler.ChannelAnswer(e.AsteriskID, e.Channel.ID); err != nil {
		return err
	}

	// set timeout for 180 sec
	if err := h.reqHandler.ChannelVariableSet(e.AsteriskID, e.Channel.ID, "TIMEOUT(absolute)", "180"); err != nil {
		return err
	}

	// continue to svc-echo
	if err := h.reqHandler.ChannelContinue(e.AsteriskID, e.Channel.ID, "svc-echo", e.Channel.Dialplan.Exten, 1, ""); err != nil {
		return err
	}

	return nil
}
