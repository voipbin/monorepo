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
	e := evt.(*ari.ChannelCreated)

	cn := channel.NewChannelByChannelCreated(e)
	cn.TMUpdate = defaultTimeStamp
	cn.TMAnswer = defaultTimeStamp
	cn.TMRinging = defaultTimeStamp
	cn.TMEnd = defaultTimeStamp
	if err := h.db.ChannelCreate(ctx, cn); err != nil {
		return err
	}

	// start channel watcher
	if err := h.reqHandler.CallV1ChannelHealth(ctx, cn.AsteriskID, cn.ID, requesthandler.DelaySecond*10, 0, 2); err != nil {
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

// EventHandlerChannelDestroyed handels ChannelDestroyed ARI event
func (h *eventHandler) EventHandlerChannelDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDestroyed)

	if err := h.db.ChannelEnd(ctx, e.Channel.ID, string(e.Timestamp), e.Cause); err != nil {
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
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

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
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
			"channel":  e.Channel.ID,
			"bridge":   e.Bridge.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if !h.db.ChannelIsExist(e.Channel.ID, defaultExistTimeout) {
		log.Error("The given channel is not in our database.")
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return fmt.Errorf("no channel found")
	}

	if !h.db.BridgeIsExist(e.Bridge.ID, defaultExistTimeout) {
		log.Error("The given bridge is not in our database.")
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return fmt.Errorf("no bridge found")
	}

	// set channel's bridge id
	if err := h.db.ChannelSetBridgeID(ctx, e.Channel.ID, e.Bridge.ID); err != nil {
		log.Errorf("Could not set the bridge id to the channel. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	// add bridge's channel id
	if err := h.db.BridgeAddChannelID(ctx, e.Bridge.ID, e.Channel.ID); err != nil {
		log.Errorf("Could not add the channel from the bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return err
	}

	br, err := h.db.BridgeGet(ctx, e.Bridge.ID)
	if err != nil {
		log.Errorf("Could not get bridge. err: %v", err)
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
			"channel":  e.Channel.ID,
			"bridge":   e.Bridge.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if !h.db.ChannelIsExist(e.Channel.ID, defaultExistTimeout) {
		log.Error("The given channel is not in our database.")
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return fmt.Errorf("no channel found")
	}

	if !h.db.BridgeIsExist(e.Bridge.ID, defaultExistTimeout) {
		log.Error("The given bridge is not in our database.")
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return fmt.Errorf("no bridge found")
	}

	// set channel's bridge id to empty
	if err := h.db.ChannelSetBridgeID(ctx, e.Channel.ID, ""); err != nil {
		log.Errorf("Could not reset the channel's bridge id. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	// remove channel from the bridge
	if err := h.db.BridgeRemoveChannelID(ctx, e.Bridge.ID, e.Channel.ID); err != nil {
		log.Errorf("Could not remove the channel from the bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	br, err := h.db.BridgeGet(ctx, e.Bridge.ID)
	if err != nil {
		log.Errorf("Could not get bridge. err: %v", err)
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

	if err := h.db.ChannelSetState(ctx, e.Channel.ID, string(e.Timestamp), e.Channel.State); err != nil {
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
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
