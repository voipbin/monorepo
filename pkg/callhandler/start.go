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
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"

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
	// set timeout for 300 sec
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho); err != nil {
		return err
	}

	// create default option for echo
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

	// start echo conference
	conf, err := h.confHandler.Start(conference.TypeEcho, c)
	if err != nil {
		return err
	}
	log.Debugf("Conference started. conf: %v", conf)

	return nil
}
