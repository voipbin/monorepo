package channelhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// HealthCheck checks the given channel is still vaild
func (h *channelHandler) HealthCheck(ctx context.Context, channelID string, retryCount int, retryCountMax int, delay int) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "HealthCheck",
			"channel_id": channelID,
		},
	)

	// get channel
	cn, err := h.Get(ctx, channelID)
	if err != nil {
		log.Errorf("Could not get the channel from the database. channel_id: %s, err: %v", channelID, err)
		return
	}
	log = logrus.WithField("asterisk_id", cn.AsteriskID)

	// check if the channel has deleted already, we don't go further.
	if cn.TMEnd != dbhandler.DefaultTimeStamp {
		log.WithField(
			"channel", cn,
		).Debug("The channel has hungup already. Stop to health-check.")
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
	if retryCount >= retryCountMax {
		log.Info("Could not get channel info correctly. Terminating the channel.")
		return
	}

	// send another health check.
	log.Debugf("Sending health-check request. retry: %d, retry_max: %d, delay: %d", retryCount, retryCountMax, delay)
	if err := h.reqHandler.CallV1ChannelHealth(ctx, channelID, delay, retryCount, retryCountMax); err != nil {
		log.Errorf("Could not send the channel check request. err: %v", err)
		return
	}
}
