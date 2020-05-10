package arievent

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// eventHandlerChannelCreated handels ChannelCreated ARI event
func (h *eventHandler) eventHandlerChannelCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelCreated)

	cn := channel.NewChannelByChannelCreated(e)
	if err := h.db.ChannelCreate(ctx, cn); err != nil {
		return err
	}

	return nil
}

// eventHandlerChannelDestroyed handels ChannelDestroyed ARI event
func (h *eventHandler) eventHandlerChannelDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelDestroyed)

	if err := h.db.ChannelEnd(ctx, e.AsteriskID, e.Channel.ID, string(e.Timestamp), e.Cause); err != nil {
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
	if err != nil {
		return err
	}

	if err := h.svcHandler.Hangup(cn); err != nil {
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
		return err
	}

	// add bridge's channel id
	if err := h.db.BridgeAddChannelID(ctx, e.Bridge.ID, e.Channel.ID); err != nil {
		return err
	}

	return nil
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
		return err
	}

	// remove channel from the bridge
	if err := h.db.BridgeRemoveChannelID(ctx, e.Bridge.ID, e.Channel.ID); err != nil {
		return err
	}

	return nil
}

// eventHandlerChannelStateChange handels ChannelStateChange ARI event
func (h *eventHandler) eventHandlerChannelStateChange(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelStateChange)

	if err := h.db.ChannelSetState(ctx, e.AsteriskID, e.Channel.ID, string(e.Timestamp), e.Channel.State); err != nil {
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
	if err != nil {
		return err
	}

	if err := h.svcHandler.UpdateStatus(cn); err != nil {
		return err
	}

	return nil
}
