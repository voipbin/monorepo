package eventhandler

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
)

// eventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) eventHandlerPlaybackFinished(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.PlaybackFinished)

	log := log.WithFields(
		log.Fields{
			"playback": e.Playback.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
			"target":   e.Playback.TargetURI,
		})

	if strings.HasPrefix(e.Playback.TargetURI, "channel:") != true {
		// no channel info
		return nil
	}

	tmpChannelID := e.Playback.TargetURI[8:]

	// check the channel is still exists.
	// if the channel was hungup while the file is playing, the asterisk sends the playbackfinished event and channeldestroyed
	// event sequentially.
	// this makes  hard to know the channel was hungup or not using the playbackfinished event.
	// so we have to send the channelget request to check the channel still does exist.
	_, err := h.reqHandler.AstChannelGet(e.AsteriskID, tmpChannelID)
	if err != nil {
		log.Infof("Could not get the channel info from the Asterisk. Consider the channel already hungup. err: %v", err)
		return nil
	}

	cn, err := h.db.ChannelGet(ctx, tmpChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return err
	}

	return h.callHandler.ARIPlaybackFinished(cn, e.Playback.ID)
}
