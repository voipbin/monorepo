package arihandler

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// eventHandlerChannelCreated handels ChannelCreated ARI event
func (h *ariHandler) eventHandlerChannelCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelCreated)

	cn := channel.NewChannelByChannelCreated(e)
	if err := h.db.ChannelCreate(ctx, cn); err != nil {
		return err
	}

	// start channel watcher
	if err := h.reqHandler.CallChannelHealth(cn.AsteriskID, cn.ID, 10*1000, 0, 2); err != nil {
		log.WithFields(
			log.Fields{
				"asterisk": cn.AsteriskID,
				"channel":  cn.ID,
			}).Errorf("Could not start the channel water. err: %v", err)
	}
	log.WithFields(
		log.Fields{
			"asterisk": cn.AsteriskID,
			"channel":  cn.ID,
		}).Debugf("Started channel watcher.")

	return nil
}

// eventHandlerChannelDestroyed handels ChannelDestroyed ARI event
func (h *ariHandler) eventHandlerChannelDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDestroyed)

	if err := h.db.ChannelEnd(ctx, e.AsteriskID, e.Channel.ID, string(e.Timestamp), e.Cause); err != nil {
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
	if err != nil {
		return err
	}

	if err := h.callHandler.ARIChannelDestroyed(cn); err != nil {
		return err
	}

	return nil
}

// eventHandlerChannelEnteredBridge handles ChannelEnteredBridge ARI event
func (h *ariHandler) eventHandlerChannelEnteredBridge(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelEnteredBridge)

	log := log.WithFields(
		log.Fields{
			"channel":  e.Channel.ID,
			"bridge":   e.Bridge.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if h.db.ChannelIsExist(e.Channel.ID, e.AsteriskID, defaultExistTimeout) == false {
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
	if err := h.db.ChannelSetBridgeID(ctx, e.AsteriskID, e.Channel.ID, e.Bridge.ID); err != nil {
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

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
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

	return h.callHandler.ARIChannelEnteredBridge(cn, bridge)
}

// eventHandlerChannelLeftBridge handles ChannelLeftBridge ARI event
func (h *ariHandler) eventHandlerChannelLeftBridge(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelLeftBridge)

	log := log.WithFields(
		log.Fields{
			"channel":  e.Channel.ID,
			"bridge":   e.Bridge.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if h.db.ChannelIsExist(e.Channel.ID, e.AsteriskID, defaultExistTimeout) == false {
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
	if err := h.db.ChannelSetBridgeID(ctx, e.AsteriskID, e.Channel.ID, ""); err != nil {
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

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
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

	return h.callHandler.ARIChannelLeftBridge(cn, bridge)
}

// eventHandlerChannelStateChange handels ChannelStateChange ARI event
func (h *ariHandler) eventHandlerChannelStateChange(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelStateChange)

	if err := h.db.ChannelSetState(ctx, e.AsteriskID, e.Channel.ID, string(e.Timestamp), e.Channel.State); err != nil {
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
	if err != nil {
		return err
	}

	if err := h.callHandler.UpdateStatus(cn); err != nil {
		return err
	}

	return nil
}
