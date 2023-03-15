package channelhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// HealthCheck checks the given channel is still vaild
func (h *channelHandler) HealthCheck(ctx context.Context, channelID string, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "HealthCheck",
		"channel_id": channelID,
	})

	// get channel
	cn, err := h.Get(ctx, channelID)
	if err != nil {
		log.Errorf("Could not get the channel from the database. channel_id: %s, err: %v", channelID, err)
		return
	}

	// check if the channel has deleted already, we don't go further.
	if cn.TMEnd != dbhandler.DefaultTimeStamp {
		// the channel has hungup already. no need to check the health anymore
		return
	}

	// send a channel heaclth check
	_, err = h.reqHandler.AstChannelGet(ctx, cn.AsteriskID, cn.ID)
	if err != nil {
		retryCount++
	} else {
		retryCount = 0
	}

	// todo: if the retry count is bigger than 2,
	// then generate fake-ChannelDestroyed event
	if retryCount > defaultHealthMaxRetryCount {
		log.WithField("channel", cn).Info("Could not get channel info correctly. Terminating the channel.")
		return
	}

	// send health check.
	if err := h.reqHandler.CallV1ChannelHealth(ctx, channelID, defaultHealthDelay, retryCount, defaultHealthMaxRetryCount); err != nil {
		log.Errorf("Could not send the channel check request. err: %v", err)
		return
	}
}
