package eventhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

// eventHandlerChannelCreated handels ChannelCreated ARI event
func (h *eventHandler) eventHandlerChannelCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelCreated)

	cn := channel.NewChannelByChannelCreated(e)
	if err := h.db.ChannelCreate(ctx, cn); err != nil {
		return err
	}

	// start channel watcher
	if err := h.reqHandler.CallChannelHealth(cn.AsteriskID, cn.ID, requesthandler.DelaySecond*10, 0, 2); err != nil {
		log.WithFields(
			log.Fields{
				"asterisk": cn.AsteriskID,
				"channel":  cn.ID,
			}).Errorf("Could not start the channel water. err: %v", err)
		return nil
	}
	log.WithFields(
		log.Fields{
			"asterisk": cn.AsteriskID,
			"channel":  cn.ID,
		}).Debugf("Started channel watcher.")

	return nil
}

// eventHandlerChannelDestroyed handels ChannelDestroyed ARI event
func (h *eventHandler) eventHandlerChannelDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDestroyed)

	if err := h.db.ChannelEnd(ctx, e.Channel.ID, string(e.Timestamp), e.Cause); err != nil {
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		return err
	}

	if err := h.callHandler.ARIChannelDestroyed(cn); err != nil {
		return err
	}

	return nil
}

// eventHandlerChannelDtmfReceived handels ChannelDtmfReceived ARI event
func (h *eventHandler) eventHandlerChannelDtmfReceived(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDtmfReceived)

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		return err
	}

	if err := h.callHandler.ARIChannelDtmfReceived(cn, e.Digit, e.Duration); err != nil {
		return err
	}

	return nil
}

// eventHandlerChannelEnteredBridge handles ChannelEnteredBridge ARI event
func (h *eventHandler) eventHandlerChannelEnteredBridge(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelEnteredBridge)

	log := log.WithFields(
		log.Fields{
			"channel":  e.Channel.ID,
			"bridge":   e.Bridge.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if h.db.ChannelIsExist(e.Channel.ID, defaultExistTimeout) == false {
		log.Error("The given channel is not in our database.")
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return fmt.Errorf("no channel found")
	}

	if h.db.BridgeIsExist(e.Bridge.ID, defaultExistTimeout) == false {
		log.Error("The given bridge is not in our database.")
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return fmt.Errorf("no bridge found")
	}

	// set channel's bridge id
	if err := h.db.ChannelSetBridgeID(ctx, e.Channel.ID, e.Bridge.ID); err != nil {
		log.Errorf("Could not set the bridge id to the channel. err: %v", err)
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return err
	}

	// add bridge's channel id
	if err := h.db.BridgeAddChannelID(ctx, e.Bridge.ID, e.Channel.ID); err != nil {
		log.Errorf("Could not add the channel from the bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return err
	}

	bridge, err := h.db.BridgeGet(ctx, e.Bridge.ID)
	if err != nil {
		log.Errorf("Could not get bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return err
	}

	return h.confHandler.ARIChannelEnteredBridge(cn, bridge)
}

// eventHandlerChannelLeftBridge handles ChannelLeftBridge ARI event
func (h *eventHandler) eventHandlerChannelLeftBridge(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelLeftBridge)

	log := log.WithFields(
		log.Fields{
			"channel":  e.Channel.ID,
			"bridge":   e.Bridge.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if h.db.ChannelIsExist(e.Channel.ID, defaultExistTimeout) == false {
		log.Error("The given channel is not in our database.")
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return fmt.Errorf("no channel found")
	}

	if h.db.BridgeIsExist(e.Bridge.ID, defaultExistTimeout) == false {
		log.Error("The given bridge is not in our database.")
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return fmt.Errorf("no bridge found")
	}

	// set channel's bridge id to empty
	if err := h.db.ChannelSetBridgeID(ctx, e.Channel.ID, ""); err != nil {
		log.Errorf("Could not reset the channel's bridge id. err: %v", err)
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return err
	}

	// remove channel from the bridge
	if err := h.db.BridgeRemoveChannelID(ctx, e.Bridge.ID, e.Channel.ID); err != nil {
		log.Errorf("Could not remove the channel from the bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return err
	}

	bridge, err := h.db.BridgeGet(ctx, e.Bridge.ID)
	if err != nil {
		log.Errorf("Could not get bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return err
	}

	return h.confHandler.ARIChannelLeftBridge(cn, bridge)
}

// eventHandlerChannelStateChange handels ChannelStateChange ARI event
func (h *eventHandler) eventHandlerChannelStateChange(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelStateChange)

	if err := h.db.ChannelSetState(ctx, e.Channel.ID, string(e.Timestamp), e.Channel.State); err != nil {
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		return err
	}

	if err := h.callHandler.ARIChannelStateChange(cn); err != nil {
		return err
	}

	return nil
}

// eventHandlerChannelCreated handels ChannelCreated ARI event
func (h *eventHandler) eventHandlerChannelVarset(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelVarset)

	switch e.Variable {
	case "VB-CONTEXT_TYPE":
		if err := h.db.ChannelSetDataItem(ctx, e.Channel.ID, "context_type", e.Value); err != nil {
			return err
		}

	case "VB-DIRECTION":
		if err := h.db.ChannelSetDirection(ctx, e.Channel.ID, channel.Direction(e.Value)); err != nil {
			return err
		}
		// increase metric
		cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
		if err != nil {
			return err
		}
		if cn.Direction != channel.DirectionNone && cn.SIPTransport != channel.SIPTransportNone {
			promChannelTransportAndDirection.WithLabelValues(string(cn.SIPTransport), string(cn.Direction)).Inc()
		}

	case "VB-SIP_CALLID":
		if err := h.db.ChannelSetSIPCallID(ctx, e.Channel.ID, e.Value); err != nil {
			return err
		}

	case "VB-SIP_PAI":
		if err := h.db.ChannelSetDataItem(ctx, e.Channel.ID, "sip_pai", e.Value); err != nil {
			return err
		}

	case "VB-SIP_PRIVACY":
		if err := h.db.ChannelSetDataItem(ctx, e.Channel.ID, "sip_privacy", e.Value); err != nil {
			return err
		}

	case "VB-SIP_TRANSPORT":
		if err := h.db.ChannelSetSIPTransport(ctx, e.Channel.ID, channel.SIPTransport(e.Value)); err != nil {
			return err
		}
		// increase metric
		cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
		if err != nil {
			return err
		}
		if cn.Direction != channel.DirectionNone && cn.SIPTransport != channel.SIPTransportNone {
			promChannelTransportAndDirection.WithLabelValues(string(cn.SIPTransport), string(cn.Direction)).Inc()
		}

	case "VB-TYPE":
		logrus.Debugf("Setting channel's type. channel: %s, type: %s", e.Channel.ID, e.Value)
		if err := h.db.ChannelSetType(ctx, e.Channel.ID, channel.Type(e.Value)); err != nil {
			return err
		}

	default:
		return nil
	}

	return nil
}
