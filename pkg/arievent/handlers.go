package arievent

import (
	"context"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	channel "gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// eventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) eventHandlerStasisStart(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisStart)

	// set data and stasis
	data := make(map[string]interface{}, 1)
	for k, v := range e.Args {
		data[k] = v
	}
	stasis := e.Application

	// update data and stasis
	log.Debugf("Updating channel stasis. stasis: %s", stasis)
	if err := h.db.ChannelSetDataAndStasis(ctx, e.AsteriskID, e.Channel.ID, data, stasis); err != nil {
		// something went wrong. Hangup at here.
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
	if err != nil {
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}

	return h.svcHandler.Start(cn)
}

// eventHandlerStasisEnd handles StasisEnd ARI event
func (h *eventHandler) eventHandlerStasisEnd(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisEnd)

	if err := h.db.ChannelSetStasis(ctx, e.AsteriskID, e.Channel.ID, ""); err != nil {
		// nothing we can do here
		return err
	}

	return nil
}

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

func (h *eventHandler) eventHandlerBridgeCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.BridgeCreated)

	b := bridge.NewBridgeByBridgeCreated(e)
	if err := h.db.BridgeCreate(ctx, b); err != nil {
		return err
	}

	return nil
}

func (h *eventHandler) eventHandlerBridgeDestroyed(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.BridgeDestroyed)

	if err := h.db.BridgeEnd(ctx, e.AsteriskID, e.Bridge.ID, string(e.Timestamp)); err != nil {
		return err
	}

	return nil
}
