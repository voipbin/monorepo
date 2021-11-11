package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// bridgeLeftJoin handles the case which join channel left from the call bridge
func (h *callHandler) bridgeLeftJoin(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"asterisk_id": cn.AsteriskID,
		"channel_id":  cn.ID,
		"bridge_id":   br.ID,
		"call_id":     br.ReferenceID,
		"func":        "bridgeLeftJoin",
	})

	log.Debug("Hangup join channel.")
	_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)

	// set empty conference id
	if err := h.db.CallSetConferenceID(ctx, br.ReferenceID, uuid.Nil); err != nil {
		log.Errorf("Could not reset the conference for a call. err: %v", err)
		return err
	}

	// get updated call info
	c, err := h.db.CallGet(ctx, br.ReferenceID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	// send call notification
	h.notifyHandler.NotifyEvent(ctx, notifyhandler.EventTypeCallUpdated, c.WebhookURI, c)

	// check the call status
	if c.Status != call.StatusProgressing && c.Status != call.StatusDialing && c.Status != call.StatusRinging {
		log.Debugf("The call is being terminating.")
		return nil
	}

	// send a call action next
	if err := h.reqHandler.CallCallActionNext(ctx, c.ID); err != nil {
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
		"bridge_id":   br.ID,
	})

	// hang up the channel
	log.Debug("Hangup external media channel.")
	_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)

	// remove all other channels
	if len(br.ChannelIDs) == 0 {
		log.Debug("No channel left in the bridge. Deleting the bridge.")
		_ = h.reqHandler.AstBridgeDelete(ctx, br.AsteriskID, br.ID)
	} else {
		log.Debugf("Channels are still remain in the bridge. channels: %d", len(br.ChannelIDs))
		h.removeAllChannelsInBridge(ctx, br)
	}

	// we don't do anything here
	log.Debugf("The external channel has left from the call")

	return nil
}

// removeAllChannelsInBridge remove the all channels in the bridge
func (h *callHandler) removeAllChannelsInBridge(ctx context.Context, bridge *bridge.Bridge) {
	logrus.WithFields(
		logrus.Fields{
			"asterisk": bridge.AsteriskID,
			"bridge":   bridge.ID,
			"channels": bridge.ChannelIDs,
		}).Debug("Hanging up all channels in the bridge.")

	// destroy all the channels in the bridge
	for _, channelID := range bridge.ChannelIDs {
		if err := h.reqHandler.AstBridgeRemoveChannel(ctx, bridge.AsteriskID, bridge.ID, channelID); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"asterisk": bridge.AsteriskID,
					"bridge":   bridge.ID,
					"channel":  channelID,
				}).Debugf("Could not hangup the channel. err: %v", err)
		}
	}
}
