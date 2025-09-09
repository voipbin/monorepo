package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

// Hangup deletes the channel.
func (h *channelHandler) Hangup(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Hangup",
		"channel_id": id,
	})

	res, err := h.Delete(ctx, id, cause)
	if err != nil {
		log.Errorf("Could not delete the channel. err: %v", err)
		return nil, err
	}

	return res, nil
}

// HangingUp starts the hangup process
func (h *channelHandler) HangingUp(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "HangingUp",
		"channel_id": id,
		"cause":      cause,
	})

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return nil, err
	}

	if res.TMDelete < dbhandler.DefaultTimeStamp {
		// already hungup nothing to do
		log.WithField("channel", res).Debugf("The channel hungup already.")
		return res, nil
	}

	if errHangup := h.HangingUpWithAsteriskID(ctx, res.AsteriskID, res.ID, cause); errHangup != nil {
		return nil, errHangup
	}

	return res, nil
}

// HangingUpWithAsteriskID starts the hangup process
func (h *channelHandler) HangingUpWithAsteriskID(ctx context.Context, asteriskID string, id string, cause ari.ChannelCause) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "HangingUpWithAsteriskID",
		"asterisk_id": asteriskID,
		"channel_id":  id,
		"cause":       cause,
	})
	log.Debugf("Hanging up channel with asteriskID: %s, channelID: %s, cause: %d", asteriskID, id, cause)

	if errHangup := h.reqHandler.AstChannelHangup(ctx, asteriskID, id, cause, 0); errHangup != nil {
		if errors.Cause(errHangup) == requesthandler.ErrNotFound {
			// channel doesn't exist. consider it hungup already.
			return nil
		}

		log.Errorf("Could not send the hangup request. err: %v", errHangup)
		return errHangup
	}

	return nil
}

// HangingUpWithDelay starts the hangup process
func (h *channelHandler) HangingUpWithDelay(ctx context.Context, id string, cause ari.ChannelCause, delay int) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "HangingUpWithTimeout",
		"channel_id": id,
	})

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return nil, err
	}

	if res.TMDelete < dbhandler.DefaultTimeStamp {
		// already hungup nothing to do
		return nil, fmt.Errorf("already hungup")
	}

	if errHangup := h.reqHandler.AstChannelHangup(ctx, res.AsteriskID, id, cause, delay); errHangup != nil {
		log.Errorf("Could not send the hangup request. err: %v", errHangup)
		return nil, errHangup
	}

	return res, nil
}
