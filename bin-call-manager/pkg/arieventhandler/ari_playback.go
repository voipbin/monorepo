package arieventhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	ari "monorepo/bin-call-manager/models/ari"
)

// EventHandlerPlaybackStarted handles PlaybackStarted ARI event
func (h *eventHandler) EventHandlerPlaybackStarted(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.PlaybackStarted)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerPlaybackStarted",
		"event": e,
	})

	if !strings.HasPrefix(e.Playback.TargetURI, "channel:") {
		// no channel info
		return nil
	}

	// get channel id and playback id
	channelID := e.Playback.TargetURI[len("channel:"):]
	playbackID := e.Playback.ID

	_, err := h.channelHandler.UpdatePlaybackID(ctx, channelID, playbackID)
	if err != nil {
		log.Errorf("Could not update the channel's playback id. channel_id: %s, err: %v", channelID, err)
		// we've failed to set the plabyback id, but the playback is working.
		// we don't return the error here.
	}

	return nil
}

// EventHandlerPlaybackFinished handles PlaybackFinished ARI event
func (h *eventHandler) EventHandlerPlaybackFinished(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.PlaybackFinished)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerPlaybackFinished",
		"event": e,
	})

	// Extract the string after the first ':' from TargetURI
	parts := strings.SplitN(e.Playback.TargetURI, ":", 2)
	if len(parts) < 2 {
		// no channel info
		return nil
	}

	targetResource := parts[0]
	targetID := parts[1]
	switch targetResource {
	case "channel":
		cn, err := h.channelHandler.UpdatePlaybackID(ctx, targetID, "")
		if err != nil {
			log.Errorf("Could not update the channel's playback id. channel_id: %s, err: %v", targetID, err)
			// we've failed to set the plabyback id, but the playback is working.
			// we don't return the error here.
		}

		if cn.TMEnd != nil {
			log.Infof("The channel already hungup. channel_id: %s", cn.ID)
			return nil
		}

		return h.callHandler.ARIPlaybackFinished(ctx, cn, e)

	case "bridge":
		br, err := h.bridgeHandler.Get(ctx, targetID)
		if err != nil {
			log.Errorf("Could not get bridge info. err: %v", err)
			return err
		}

		if br.TMDelete != nil {
			log.Infof("The bridge already deleted. bridge_id: %s", br.ID)
			return nil
		}

		return h.externalmediaHandler.ARIPlaybackFinished(ctx, br, e)

	default:
		return fmt.Errorf("unsupported target resource: %s", targetResource)

	}
}
