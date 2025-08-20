package externalmediahandler

import (
	"context"
	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/playback"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *externalMediaHandler) ARIPlaybackFinished(ctx context.Context, cn *channel.Channel, e *ari.PlaybackFinished) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ARIPlaybackFinished",
		"channel": cn,
		"event":   e,
	})

	id := uuid.FromStringOrNil(strings.TrimPrefix(e.Playback.ID, playback.IDPrefixExternalMedia))
	tmp, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get external media with id: %s", id)
	}

	if tmp.Status != externalmedia.StatusRunning {
		// nothing to do
		return nil
	}

	if errPlay := h.channelHandler.Play(ctx, cn.ID, e.Playback.ID, []string{defaultSilencePlaybackMedia}, "", 0, 0); errPlay != nil {
		return errors.Wrapf(errPlay, "could not start silence playback for channel_id: %s", cn.ID)
	}
	log.Debugf("Started silence playback for the channel. channel_id: %s", cn.ID)

	return nil
}
