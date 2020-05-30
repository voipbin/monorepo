package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// ARIChannelEnteredBridge is called when the channel handler received ChannelEnteredBridge.
func (h *callHandler) ARIChannelEnteredBridge(cn *channel.Channel, bridge *bridge.Bridge) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
			"bridge":   bridge.ID,
		})

	cnContext := cn.Data["CONTEXT"]
	if cnContext == nil || cnContext != contextIncomingCall {
		log.Debug("The channel is not for the call.")
		return nil
	}

	// get call
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not find a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	if err := h.db.CallSetConferenceID(ctx, c.ID, bridge.ConferenceID); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set the conference for a call. call: %s, conference: %s, err: %v", c.ID, bridge.ConferenceID, err)
	}

	if err := h.confHandler.Joined(bridge.ConferenceID, c.ID); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not joined to the conference. call: %s, conference: %s, err: %v", c.ID, bridge.ConferenceID, err)
	}

	log.WithFields(
		logrus.Fields{
			"call":       c.ID,
			"conference": bridge.ConferenceID,
		}).Debugf("The call has entered to the conference.")

	return nil
}

// ARIChannelLeftBridge is called when the channel handler received ChannelLeftBridge.
func (h *callHandler) ARIChannelLeftBridge(cn *channel.Channel, bridge *bridge.Bridge) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
			"bridge":   bridge.ID,
		})

	// this channel is not for the call.
	if getContextType(cn.Data["CONTEXT"]) != contextTypeCall {
		return h.confHandler.ARIChannelLeftBridge(cn, bridge)
	}

	cnContext := cn.Data["CONTEXT"]
	if cnContext == nil || cnContext != contextIncomingCall {
		log.Debug("The channel is not for the call.")
		return nil
	}

	// get call
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not find a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// set empty conference id
	if err := h.db.CallSetConferenceID(ctx, c.ID, uuid.Nil); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not reset the conference for a call. call: %s, conference: %s, err: %v", c.ID, bridge.ConferenceID, err)
	}

	// notice to the conference
	if err := h.confHandler.Leaved(c.ConfID, c.ID); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not leaved from the conference. call: %s, conference: %s, err: %v", c.ID, bridge.ConferenceID, err)
	}

	// do next action
	return h.ActionNext(c)
}

// ARIStasisStart is called when the channel handler received StasisStart.
func (h *callHandler) ARIStasisStart(cn *channel.Channel) error {
	contextType := getContextType(cn.Data["CONTEXT"])
	switch contextType {
	case contextTypeConference:
		return h.confHandler.ARIStasisStart(cn)
	default:
		return h.Start(cn)
	}
}

// ARIChannelDestroyed handles ChannelDestroyed ARI event
func (h *callHandler) ARIChannelDestroyed(cn *channel.Channel) error {
	contextType := getContextType(cn.Data["CONTEXT"])
	switch contextType {
	case contextTypeConference:
		return h.confHandler.ARIStasisStart(cn)
	case contextTypeCall:
		return h.Hangup(cn)
	default:
		return nil
	}
}

// ARIChannelDtmfReceived handles ChannelDtmfReceived ARI event
func (h *callHandler) ARIChannelDtmfReceived(cn *channel.Channel, digit string, duration int) error {

	// support pjsip type only for now.
	if cn.Tech != channel.TechPJSIP {
		return nil
	}

	if err := h.DTMFReceived(cn, digit, duration); err != nil {
		return err
	}

	return nil
}
