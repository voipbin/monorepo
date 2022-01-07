package callhandler

import (
	"context"

	"github.com/sirupsen/logrus"
)

// ChannelHealthCheck checks the given channel is still vaild
func (h *callHandler) ChannelHealthCheck(ctx context.Context, asteriskID string, channelID string, retryCount int, retryCountMax int, delay int) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "ChannelHealthCheck",
			"asterisk_id": asteriskID,
			"channel_id":  channelID,
		},
	)

	// get channel
	channel, err := h.db.ChannelGet(ctx, channelID)
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"asterisk": asteriskID,
				"channel":  channelID,
			}).Errorf("Could not get the channel from the database. err: %v", err)
	}

	// check if the channel has deleted already, we don't go further.
	if channel.TMEnd != "" {
		log.WithField(
			"channel", channel,
		).Debug("The channel has hungup already. Stop to health-check.")
		return
	}

	// send a channel heaclth check
	_, err = h.reqHandler.AstChannelGet(ctx, asteriskID, channelID)
	if err != nil {
		retryCount++
	} else {
		retryCount = 0
	}

	// todo: if the retry count is bigger than 2,
	// then generate fake-ChannelDestroyed event
	if retryCount >= retryCountMax {
		logrus.WithFields(
			logrus.Fields{
				"asterisk": asteriskID,
				"channel":  channelID,
			}).Info("Could not get channel info correctly. Terminating the channel.")
		return
	}

	// send another health check.
	log.Debugf("Sending health-check request. retry: %d, retry_max: %d, delay: %d", retryCount, retryCountMax, delay)
	if err := h.reqHandler.CMV1ChannelHealth(ctx, asteriskID, channelID, delay, retryCount, retryCountMax); err != nil {
		log.Errorf("Could not send the channel check request. err: %v", err)
		return
	}
}
