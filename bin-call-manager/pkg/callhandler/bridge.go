package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

// bridgeLeftJoin handles the case which join channel left from the call bridge
func (h *callHandler) bridgeLeftJoin(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "bridgeLeftJoin",
		"channel": cn,
		"bridge":  br,
	})

	log.Debug("Hangup join channel.")
	_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNormalClearing)

	// set empty conference id
	if err := h.db.CallSetConfbridgeID(ctx, br.ReferenceID, uuid.Nil); err != nil {
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
	h.notifyHandler.PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallUpdated, c)

	// check the call status
	if c.Status != call.StatusProgressing && c.Status != call.StatusDialing && c.Status != call.StatusRinging {
		log.Debugf("The call is being terminating.")
		return nil
	}

	// send a call action next
	if err := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); err != nil {
		log.Debugf("Could not send the call action next request. err: %v", err)
		return err
	}

	return nil
}

// bridgeLeftExternal handles the case which external channel left from the call bridge
func (h *callHandler) bridgeLeftExternal(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "bridgeLeftExternal",
		"channel_id": cn.ID,
		"bridge_id":  br.ID,
	})

	// hang up the channel
	log.Debug("Hangup external media channel.")
	_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNormalClearing)

	// remove all other channels
	if len(br.ChannelIDs) == 0 {
		log.Debug("No channel left in the bridge. Deleting the bridge.")
		_ = h.bridgeHandler.Destroy(ctx, br.ID)
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
	log := logrus.WithFields(logrus.Fields{
		"func":   "removeAllChannelsInBridge",
		"bridge": bridge,
	})
	log.Debug("Hanging up all channels in the bridge.")

	// destroy all the channels in the bridge
	for _, channelID := range bridge.ChannelIDs {
		log.Debugf("Kicking out the channel from the bridge. channel_id: %s", channelID)
		if errKick := h.bridgeHandler.ChannelKick(ctx, bridge.ID, channelID); errKick != nil {
			log.Debugf("Could not hangup the channel. channel_id: %s, err: %v", channelID, errKick)
		}
	}
}
