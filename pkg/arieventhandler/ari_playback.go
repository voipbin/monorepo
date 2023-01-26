package arieventhandler

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// EventHandlerPlaybackStarted handles PlaybackStarted ARI event
func (h *eventHandler) EventHandlerPlaybackStarted(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.PlaybackStarted)

	log := log.WithFields(
		log.Fields{
			"func":        "eventHandlerPlaybackStarted",
			"playback_id": e.Playback.ID,
			"asterisk_id": e.AsteriskID,
			"stasis_name": e.Application,
			"target":      e.Playback.TargetURI,
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

	log := log.WithFields(
		log.Fields{
			"func":        "eventHandlerPlaybackFinished",
			"playback_id": e.Playback.ID,
			"asterisk_id": e.AsteriskID,
			"stasis_name": e.Application,
			"target":      e.Playback.TargetURI,
		})

	if !strings.HasPrefix(e.Playback.TargetURI, "channel:") {
		// no channel info
		return nil
	}

	channelID := e.Playback.TargetURI[len("channel:"):]
	cn, err := h.channelHandler.UpdatePlaybackID(ctx, channelID, "")
	if err != nil {
		log.Errorf("Could not update the channel's playback id. channel_id: %s, err: %v", channelID, err)
		// we've failed to set the plabyback id, but the playback is working.
		// we don't return the error here.
	}

	if cn.TMEnd < dbhandler.DefaultTimeStamp {
		log.Infof("The channel already hungup. channel_id: %s", cn.ID)
		return nil
	}

	return h.callHandler.ARIPlaybackFinished(ctx, cn, e.Playback.ID)
}
