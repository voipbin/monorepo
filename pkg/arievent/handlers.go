package arievent

import (
	"context"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	channel "gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// eventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) eventHandlerStasisStart(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisStart)

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
	if err != nil {
		h.reqHandler.ChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}

	for k, v := range e.Args {
		cn.Data[k] = v
	}
	if err := h.db.ChannelSetData(ctx, e.AsteriskID, e.Channel.ID, string(e.Timestamp), cn.Data); err != nil {
		h.reqHandler.ChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}
	cn.TMUpdate = string(e.Timestamp)

	return h.svcHandler.Start(cn)
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
