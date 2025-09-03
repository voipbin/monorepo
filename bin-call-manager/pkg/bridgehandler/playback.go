package bridgehandler

import (
	"context"
	"fmt"
	"monorepo/bin-call-manager/pkg/dbhandler"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Play plays the given medias to the bridge.
func (h *bridgeHandler) Play(
	ctx context.Context,
	id string,
	playbackID string,
	medias []string,
	language string,
	offsetms int,
	skipms int,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Play",
		"bridge_id":   id,
		"playback_id": playbackID,
	})

	br, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return errors.Wrap(err, "could not get bridge info")
	}

	if br.TMDelete < dbhandler.DefaultTimeStamp {
		log.Errorf("The bridge has hungup already.")
		return fmt.Errorf("the bridge has hungup already")
	}

	tmp, err := h.reqHandler.AstBridgePlay(ctx, br.AsteriskID, br.ID, medias, language, offsetms, skipms, playbackID)
	if err != nil {
		return errors.Wrapf(err, "could not play the media on the bridge. media: %v", medias)
	}
	log.WithField("playback", tmp).Debugf("Playback requested successfully")

	return nil
}
