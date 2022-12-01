package arieventhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	ari "gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// EventHandlerChannelCreated handels ChannelCreated ARI event
func (h *eventHandler) EventHandlerChannelCreated(ctx context.Context, evt interface{}) error {
	log := logrus.WithField("func", "EventHandlerChannelCreated")

	e := evt.(*ari.ChannelCreated)
	log = log.WithFields(logrus.Fields{
		"channel_id":  e.Channel.ID,
		"asterisk_id": e.AsteriskID,
	})

	tech := channel.GetTech(e.Channel.Name)
	cn, err := h.channelHandler.Create(
		ctx,

		e.Channel.ID,
		e.AsteriskID,
		e.Channel.Name,
		channel.TypeNone,
		tech,

		"",                       // sipCallID
		channel.SIPTransportNone, // sipTransport

		e.Channel.Caller.Name,
		e.Channel.Caller.Number,
		"",                       // destinationName
		e.Channel.Dialplan.Exten, // destinationNumber

		e.Channel.State,
		map[string]interface{}{},

		"",
		map[string]string{},

		"",
		"",
		"",
		ari.ChannelCauseUnknown,
		channel.DirectionNone,
	)
	if err != nil {
		log.Errorf("Could not create a channel info. channel_id: %s, err: %v", cn.ID, err)
		return err
	}
	log.WithField("channel", cn).Debugf("Created a channel info. channel_id: %s", cn.ID)

	// start channel watcher
	if err := h.reqHandler.CallV1ChannelHealth(ctx, cn.ID, requesthandler.DelaySecond*10, 0, 2); err != nil {
		logrus.Errorf("Could not start the channel water. err: %v", err)
		return nil
	}
	log.Debugf("Started channel watcher.")

	return nil
}

// EventHandlerChannelDestroyed handels ChannelDestroyed ARI event
func (h *eventHandler) EventHandlerChannelDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDestroyed)

	cn, err := h.channelHandler.Delete(ctx, e.Channel.ID, e.Cause)
	if err != nil {
		return err
	}

	if err := h.callHandler.ARIChannelDestroyed(ctx, cn); err != nil {
		return err
	}

	return nil
}

// EventHandlerChannelDtmfReceived handels ChannelDtmfReceived ARI event
func (h *eventHandler) EventHandlerChannelDtmfReceived(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDtmfReceived)

	cn, err := h.channelHandler.Get(ctx, e.Channel.ID)
	if err != nil {
		return err
	}

	if err := h.callHandler.ARIChannelDtmfReceived(ctx, cn, e.Digit, e.Duration); err != nil {
		return err
	}

	return nil
}

// EventHandlerChannelEnteredBridge handles ChannelEnteredBridge ARI event
func (h *eventHandler) EventHandlerChannelEnteredBridge(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelEnteredBridge)

	log := log.WithFields(
		log.Fields{
			"func":        "EventHandlerChannelEnteredBridge",
			"channel_id":  e.Channel.ID,
			"bridge_id":   e.Bridge.ID,
			"asterisk_id": e.AsteriskID,
			"stasis":      e.Application,
		})

	cn, err := h.channelHandler.UpdateBridgeID(ctx, e.Channel.ID, e.Bridge.ID)
	if err != nil {
		log.Errorf("Could not set the bridge id to the channel. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	_, err = h.bridgeHandler.GetWithTimeout(ctx, e.Bridge.ID, defaultExistTimeout)
	if err != nil {
		log.Errorf("Could not get the bridge within timeout. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	br, err := h.bridgeHandler.AddChannelID(ctx, e.Bridge.ID, e.Channel.ID)
	if err != nil {
		log.Errorf("Could not set the bridge id to the channel. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	if cn.Type == channel.TypeConfbridge {
		return h.confbridgeHandler.ARIChannelEnteredBridge(ctx, cn, br)
	}

	return nil
}

// EventHandlerChannelLeftBridge handles ChannelLeftBridge ARI event
func (h *eventHandler) EventHandlerChannelLeftBridge(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelLeftBridge)

	log := log.WithFields(
		log.Fields{
			"func":        "EventHandlerChannelLeftBridge",
			"channel_id":  e.Channel.ID,
			"bridge_id":   e.Bridge.ID,
			"asterisk_id": e.AsteriskID,
			"stasis":      e.Application,
		})

	cn, err := h.channelHandler.UpdateBridgeID(ctx, e.Channel.ID, "")
	if err != nil {
		log.Errorf("Could not reset the channel's bridge id. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	tmp, err := h.bridgeHandler.GetWithTimeout(ctx, e.Bridge.ID, defaultExistTimeout)
	if err != nil {
		log.Error("The given bridge is not in our database.")
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return fmt.Errorf("no bridge found")
	}
	log.WithField("bridge", tmp).Debugf("Found bridge info. bridge_id: %s", tmp.ID)

	br, err := h.bridgeHandler.RemoveChannelID(ctx, tmp.ID, e.Channel.ID)
	if err != nil {
		log.Errorf("Could not remove the channel from the bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	switch br.ReferenceType {
	case bridge.ReferenceTypeCall, bridge.ReferenceTypeCallSnoop:
		return h.callHandler.ARIChannelLeftBridge(ctx, cn, br)

	case bridge.ReferenceTypeConfbridge, bridge.ReferenceTypeConfbridgeSnoop:
		return h.confbridgeHandler.ARIChannelLeftBridge(ctx, cn, br)

	default:
		log.WithField("event", e).Error("Could not find correct event handler.")
		return nil
	}
}

// EventHandlerChannelStateChange handels ChannelStateChange ARI event
func (h *eventHandler) EventHandlerChannelStateChange(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelStateChange)

	cn, err := h.channelHandler.UpdateState(ctx, e.Channel.ID, e.Channel.State)
	if err != nil {
		return err
	}

	if err := h.callHandler.ARIChannelStateChange(ctx, cn); err != nil {
		return err
	}

	return nil
}

// EventHandlerChannelVarset handels ChannelVarset ARI event
func (h *eventHandler) EventHandlerChannelVarset(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelVarset)

	switch e.Variable {
	case "VB-CONTEXT_TYPE":
		if err := h.channelHandler.SetDataItem(ctx, e.Channel.ID, "context_type", e.Value); err != nil {
			return err
		}

	case "VB-DIRECTION":
		if err := h.channelHandler.SetDirection(ctx, e.Channel.ID, channel.Direction(e.Value)); err != nil {
			return err
		}

	case "VB-SIP_CALLID":
		if err := h.channelHandler.SetSIPCallID(ctx, e.Channel.ID, e.Value); err != nil {
			return err
		}

	case "VB-SIP_PAI":
		if err := h.channelHandler.SetDataItem(ctx, e.Channel.ID, "sip_pai", e.Value); err != nil {
			return err
		}

	case "VB-SIP_PRIVACY":
		if err := h.channelHandler.SetDataItem(ctx, e.Channel.ID, "sip_privacy", e.Value); err != nil {
			return err
		}

	case "VB-SIP_TRANSPORT":
		if err := h.channelHandler.SetSIPTransport(ctx, e.Channel.ID, channel.SIPTransport(e.Value)); err != nil {
			return err
		}

	case "VB-TYPE":
		logrus.Debugf("Setting channel's type. channel: %s, type: %s", e.Channel.ID, e.Value)
		if err := h.channelHandler.SetType(ctx, e.Channel.ID, channel.Type(e.Value)); err != nil {
			return err
		}

	default:
		return nil
	}

	return nil
}
