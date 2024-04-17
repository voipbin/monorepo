package bridgehandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ChannelKick kicks the channel from the bridge
func (h *bridgeHandler) ChannelKick(ctx context.Context, id string, channelID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ChannelKick",
		"bridge_id":  id,
		"channel_id": channelID,
	})

	// get bridge
	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return errors.Wrap(err, "could not get bridge info")
	}

	if errRemove := h.reqHandler.AstBridgeRemoveChannel(ctx, tmp.AsteriskID, tmp.ID, channelID); errRemove != nil {
		log.Errorf("Could not remove the channel from the bridge. err: %v", errRemove)
		return errors.Wrap(errRemove, "could not remove the channel from the bridge")
	}

	return nil
}

// ChannelJoin joins the channel from the bridge
func (h *bridgeHandler) ChannelJoin(ctx context.Context, id string, channelID string, role string, absorbDTMF bool, mute bool) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ChannelKick",
		"bridge_id":  id,
		"channel_id": channelID,
	})

	// get bridge
	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return errors.Wrap(err, "could not get bridge info")
	}

	if errAdd := h.reqHandler.AstBridgeAddChannel(ctx, tmp.AsteriskID, tmp.ID, channelID, role, absorbDTMF, mute); errAdd != nil {
		log.Errorf("Could not add the channel from the bridge. err: %v", errAdd)
		return errors.Wrap(errAdd, "could not add the channel from the bridge")
	}

	return nil
}
