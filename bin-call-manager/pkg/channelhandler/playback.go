package channelhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// PlaybackStop stops the given channel's playback
func (h *channelHandler) PlaybackStop(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PlaybackStop",
		"channel_id": id,
	})

	cn, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		log.Errorf("The channel has hungup already.")
		return fmt.Errorf("the channel has hungup already")
	}

	if cn.PlaybackID == "" {
		// no playback is playing. nothing to do
		return nil
	}

	if errStop := h.reqHandler.AstPlaybackStop(ctx, cn.AsteriskID, cn.PlaybackID); errStop != nil {
		log.Errorf("Could not stop the playback. err: %v", errStop)
		return errors.Wrap(errStop, "could not stop the playback")
	}

	return nil
}

// Play plays the given medias to the channel.
func (h *channelHandler) Play(ctx context.Context, id string, actionID uuid.UUID, medias []string, language string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Play",
		"channel_id": id,
		"action_id":  actionID,
	})

	cn, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		log.Errorf("The channel has hungup already.")
		return fmt.Errorf("the channel has hungup already")
	}

	if err := h.reqHandler.AstChannelPlay(ctx, cn.AsteriskID, cn.ID, actionID, medias, ""); err != nil {
		log.Errorf("Could not play the media. media: %v, err: %v", medias, err)
		return errors.Wrap(err, "could not play the media")
	}

	return nil
}
