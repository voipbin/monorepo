package arievent

import (
	"context"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	channel "gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// eventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) eventHandlerStasisStart(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisStart)

	return h.svcHandler.StasisStart(e)
}

// eventHandlerChannelCreated handels ChannelCreated ARI event
func (h *eventHandler) eventHandlerChannelCreated(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ChannelCreated)

	tech := getTech(e.Channel.Name)
	tmpChannel := channel.Channel{
		AsteriskID: e.AsteriskID,
		ID:         e.Channel.ID,
		Name:       e.Channel.Name,
		Tech:       tech,

		SourceName:        e.Channel.Caller.Name,
		SourceNumber:      e.Channel.Caller.Number,
		DestinationNumber: e.Channel.Dialplan.Exten,

		State: string(e.Channel.State),

		TMCreate: string(e.Timestamp),
	}

	if err := h.db.ChannelCreate(ctx, tmpChannel); err != nil {
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

	return nil
}
