package eventhandler

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/ari"
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
	cn, err := h.db.ChannelGet(ctx, tmpChannelID)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return err
	}

	return h.callHandler.ARIPlaybackFinished(cn, e.Playback.ID)
}
