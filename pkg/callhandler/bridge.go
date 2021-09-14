package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// bridgeLeftJoin handles the case which join channel left from the call bridge
func (h *callHandler) bridgeLeftJoin(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"asterisk_id": cn.AsteriskID,
		"channel_id":  cn.ID,
	})

	log.Debug("Hangup join channel.")
	h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)

	// get call
	c, err := h.db.CallGet(ctx, br.ReferenceID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	if c.Status != call.StatusProgressing {
		log.Debugf("The call is being terminating.")
		return nil
	}

	// remove the call from the conference
	if err := h.db.ConferenceRemoveCallID(ctx, c.ConfID, c.ID); err != nil {
		log.Errorf("Could not remove the call id from the conference. err: %v", err)
		return err
	}

	// set empty conference id
	if err := h.db.CallSetConferenceID(ctx, c.ID, uuid.Nil); err != nil {
		log.Errorf("Could not reset the conference for a call. err: %v", err)
		return err
	}

	// send a call action next
	if err := h.reqHandler.CallCallActionNext(c.ID); err != nil {
		log.Debugf("Could not send the call action next request. err: %v", err)
		return err
	}

	return nil
}

// bridgeLeftExternal handles the case which external channel left from the call bridge
func (h *callHandler) bridgeLeftExternal(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"asterisk_id": cn.AsteriskID,
		"channel_id":  cn.ID,
	})

	// hang up the channel
	log.Debug("Hangup external media channel.")
	h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)

	// remove all other channels
	if len(br.ChannelIDs) == 0 {
		h.reqHandler.AstBridgeDelete(br.AsteriskID, br.ID)
	} else {
		h.removeAllChannelsInBridge(br)
	}

	// we don't do anything here
	log.Debugf("The external channel has left from the call")

	return nil
}

// removeAllChannelsInBridge remove the all channels in the bridge
func (h *callHandler) removeAllChannelsInBridge(bridge *bridge.Bridge) {
	logrus.WithFields(
		logrus.Fields{
			"asterisk": bridge.AsteriskID,
			"bridge":   bridge.ID,
			"channels": bridge.ChannelIDs,
		}).Debug("Hanging up all channels in the bridge.")

	// destroy all the channels in the bridge
	for _, channelID := range bridge.ChannelIDs {
		if err := h.reqHandler.AstBridgeRemoveChannel(bridge.AsteriskID, bridge.ID, channelID); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"asterisk": bridge.AsteriskID,
					"bridge":   bridge.ID,
					"channel":  channelID,
				}).Debugf("Could not hangup the channel. err: %v", err)
		}
	}
}
