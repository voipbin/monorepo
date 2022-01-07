package arieventhandler

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// EventHandlerPlaybackStarted handles PlaybackStarted ARI event
func (h *eventHandler) EventHandlerPlaybackStarted(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.PlaybackStarted)

	log := log.WithFields(
		log.Fields{
			"func":     "eventHandlerPlaybackStarted",
			"playback": e.Playback.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
			"target":   e.Playback.TargetURI,
		})

	if !strings.HasPrefix(e.Playback.TargetURI, "channel:") {
		// no channel info
		return nil
	}

	tmpChannelID := e.Playback.TargetURI[8:]
	playbackID := e.Playback.ID

	// check the channel is still exists.
	// if the channel was hungup while the file is playing, the asterisk sends the playbackfinished event and channeldestroyed
	// event sequentially.
	// this makes  hard to know the channel was hungup or not using the playbackfinished event.
	// so we have to send the channelget request to check the channel still does exist.
	_, err := h.reqHandler.AstChannelGet(ctx, e.AsteriskID, tmpChannelID)
	if err != nil {
		log.Infof("Could not get the channel info from the Asterisk. Consider the channel already hungup. err: %v", err)
		return nil
	}

	if err := h.db.ChannelSetPlaybackID(ctx, tmpChannelID, playbackID); err != nil {
		log.Errorf("Could not set the channel's playback id. err: %v", err)
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
			"func":     "eventHandlerPlaybackFinished",
			"playback": e.Playback.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
			"target":   e.Playback.TargetURI,
		})

	if !strings.HasPrefix(e.Playback.TargetURI, "channel:") {
		// no channel info
		return nil
	}

	tmpChannelID := e.Playback.TargetURI[8:]
	if err := h.db.ChannelSetPlaybackID(ctx, tmpChannelID, ""); err != nil {
		log.Errorf("Could not set the channel's playback id. err: %v", err)
		// we've failed to set the plabyback id, but this is ok.
		// we don't return the error here.
	}

	// check the channel is still exists.
	// if the channel was hungup while the file is playing, the asterisk sends the playbackfinished event and channeldestroyed
	// event sequentially.
	// this makes  hard to know the channel was hungup or not using the playbackfinished event.
	// so we have to send the channelget request to check the channel still does exist.
	_, err := h.reqHandler.AstChannelGet(ctx, e.AsteriskID, tmpChannelID)
	if err != nil {
		log.Infof("Could not get the channel info from the Asterisk. Consider the channel already hungup. err: %v", err)
		return nil
	}

	cn, err := h.db.ChannelGet(ctx, tmpChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return err
	}

	return h.callHandler.ARIPlaybackFinished(ctx, cn, e.Playback.ID)
}
