package arieventhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	ari "monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// EventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) EventHandlerStasisStart(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisStart)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerStasisStart",
		"event": e,
	})

	tmp, err := h.channelHandler.Get(ctx, e.Channel.ID)
	if err != nil {
		log.Error("The given channel is not in our database.")
		_ = h.channelHandler.HangingUpWithAsteriskID(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return errors.Wrap(err, "no channel found")
	}
	log.WithField("channel", tmp).Debugf("Found channel info. channel_id: %s", tmp.ID)

	cn, err := h.channelHandler.ARIStasisStart(ctx, e)
	if err != nil {
		log.Errorf("The channel handler could not handle the event. err: %v", err)
		return errors.Wrap(err, "the channel handler could not handle the event")
	}
	log.WithField("channel", cn).Debugf("Updated channel info. channel_id: %s", cn.ID)

	// execute the context type handler
	contextType := h.getChannelContextType(cn)
	switch contextType {
	case channel.ContextTypeCall:
		err = h.callHandler.ARIStasisStart(ctx, cn)

	case channel.ContextTypeConference:
		err = h.confbridgeHandler.ARIStasisStart(ctx, cn)

	default:
		log.Errorf("Could not find context type handler. context_type: %s", cn.StasisData[channel.StasisDataTypeContextType])
		err = fmt.Errorf("could not find context type handler. context_type: %s", cn.StasisData[channel.StasisDataTypeContextType])
	}
	if err != nil {
		log.Errorf("Could not handle the event correctly. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return err
	}
	return nil
}

// EventHandlerStasisEnd handles StasisEnd ARI event
func (h *eventHandler) EventHandlerStasisEnd(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisEnd)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerStasisEnd",
		"event": e,
	})

	_, err := h.channelHandler.UpdateStasisName(ctx, e.Channel.ID, "")
	if err != nil {
		// could not update the channel's stasis name to empty.
		// but we don't do anything here because it's not critical.
		// and nothing we can do here because it's already end of stasis.
		log.Errorf("Could not update the stasis name. err: %v", err)
		return err
	}

	return nil
}
