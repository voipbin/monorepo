package arieventhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	ari "monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/channel"
)

// EventHandlerChannelCreated handels ChannelCreated ARI event
func (h *eventHandler) EventHandlerChannelCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelCreated)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerChannelCreated",
		"event": e,
	})

	tech := channel.GetTech(e.Channel.Name)
	cn, err := h.channelHandler.Create(
		ctx,

		e.Channel.ID,
		e.AsteriskID,
		e.Channel.Name,
		channel.TypeNone,
		tech,

		e.Channel.Caller.Name,
		e.Channel.Caller.Number,
		"",                       // destinationName
		e.Channel.Dialplan.Exten, // destinationNumber

		e.Channel.State,
	)
	if err != nil {
		log.Errorf("Could not create a channel info. channel_id: %s, err: %v", cn.ID, err)
		return err
	}
	log.WithField("channel", cn).Debugf("Created a channel info. channel_id: %s", cn.ID)

	return nil
}

// EventHandlerChannelDestroyed handels ChannelDestroyed ARI event
func (h *eventHandler) EventHandlerChannelDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDestroyed)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerChannelDestroyed",
		"event": e,
	})

	cn, err := h.channelHandler.Delete(ctx, e.Channel.ID, e.Cause)
	if err != nil {
		log.Errorf("Could not delete the channel info. err: %v", err)
		return err
	}

	switch cn.Type {
	case channel.TypeCall:
		if errDestroy := h.callHandler.ARIChannelDestroyed(ctx, cn); errDestroy != nil {
			log.Errorf("Could not delete the related call info. err: %v", errDestroy)
			return errDestroy
		}

	case channel.TypeConfbridge, channel.TypeJoin, channel.TypeExternal, channel.TypeRecording, channel.TypeApplication:
		// we don't do anything at here.
		return nil

	default:
		log.WithField("channel", cn).Errorf("Unsupported channel type. type: %v", cn.Type)
		return nil
	}

	return nil
}

// EventHandlerChannelDtmfReceived handels ChannelDtmfReceived ARI event
func (h *eventHandler) EventHandlerChannelDtmfReceived(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDtmfReceived)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerChannelDtmfReceived",
		"event": e,
	})

	cn, err := h.channelHandler.Get(ctx, e.Channel.ID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
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

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerChannelEnteredBridge",
		"event": e,
	})

	cn, err := h.channelHandler.UpdateBridgeID(ctx, e.Channel.ID, e.Bridge.ID)
	if err != nil {
		log.Errorf("Could not set the bridge id to the channel. err: %v", err)
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

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerChannelLeftBridge",
		"event": e,
	})

	cn, err := h.channelHandler.UpdateBridgeID(ctx, e.Channel.ID, "")
	if err != nil {
		log.Errorf("Could not reset the channel's bridge id. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return err
	}

	br, err := h.bridgeHandler.RemoveChannelID(ctx, e.Bridge.ID, e.Channel.ID)
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

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerChannelStateChange",
		"event": e,
	})

	cn, err := h.channelHandler.ARIChannelStateChange(ctx, e)
	if err != nil {
		log.Errorf("Could not update the channel state. err: %v", err)
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

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerChannelVarset",
		"event": e,
	})

	if errSet := h.channelHandler.SetDataItem(ctx, e.Channel.ID, e.Variable, e.Value); errSet != nil {
		log.Errorf("Could not set the variable. err: %v", errSet)
		return errors.Wrap(errSet, "could not set the variable")
	}

	return nil
}
