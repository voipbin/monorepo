package arieventhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// EventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) EventHandlerStasisStart(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisStart)

	log := log.WithFields(
		log.Fields{
			"func":        "EventHandlerStasisStart",
			"asterisk_id": e.AsteriskID,
			"channel_id":  e.Channel.ID,
			"stasis_name": e.Application,
			"stasis_data": e.Args,
		})

	tmp, err := h.channelHandler.Get(ctx, e.Channel.ID)
	if err != nil {
		log.Error("The given channel is not in our database.")
		_ = h.channelHandler.HangingUpWithAsteriskID(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return errors.Wrap(err, "no channel found")
	}
	log.WithField("channel", tmp).Debugf("Found channel info. channel_id: %s", tmp.ID)

	// get stasis name and parse the stasis data
	stasisName := e.Application
	stasisData := e.Args

	// update channel's stasis name and stasis data
	cn, err := h.channelHandler.UpdateStasisNameAndStasisData(ctx, e.Channel.ID, stasisName, stasisData)
	if err != nil {
		log.Errorf("Could not update the channel's stasis name and stasis data. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}

	contextType := getContextType(stasisData["context"])
	switch contextType {
	case contextTypeCall:
		err = h.callHandler.ARIStasisStart(ctx, cn)

	case contextTypeConfbridge:
		err = h.confbridgeHandler.ARIStasisStart(ctx, cn)

	default:
		log.Errorf("Could not find context type handler. context_type: %s", contextType)
		err = fmt.Errorf("could not find context type handler. context_type: %s", contextType)
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

	_, err := h.channelHandler.UpdateStasisName(ctx, e.Channel.ID, "")
	if err != nil {
		// could not update the channel's stasis name to empty.
		// but we don't do anything here because it's not critical.
		// and nothing we can do here because it's already end of stasis.
		return err
	}

	return nil
}
