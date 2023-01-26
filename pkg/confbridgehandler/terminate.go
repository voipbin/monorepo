package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Terminate is terminating the conference
func (h *confbridgeHandler) Terminate(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Terminate",
		"confbridge_id": id,
	})
	log.Debug("Terminating the confbridge.")

	// get confbridge
	cb, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return err
	}

	if errHangup := h.destroyBridge(ctx, cb.BridgeID); errHangup != nil {
		log.Errorf("Could not hangup the channels from the bridge. err: %v", errHangup)
		return errHangup
	}

	// update conference status to terminated
	if err := h.db.ConfbridgeDelete(ctx, id); err != nil {
		log.Errorf("Could not terminate the confbridge. err: %v", err)
		return err
	}
	promConfbridgeCloseTotal.Inc()

	// notify conference deleted event
	tmpCB, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge info. err: %v", err)
		return nil
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeDeleted, tmpCB)

	return nil
}

// destroyBridge hangs up all the channels from the bridge and destroy it.
func (h *confbridgeHandler) destroyBridge(ctx context.Context, bridgeID string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "destroyBridge",
			"bridge_id": bridgeID,
		})

	if !h.bridgeHandler.IsExist(ctx, bridgeID) {
		return nil
	}

	// hang up all the channels in the bridge
	br, err := h.bridgeHandler.Get(ctx, bridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return err
	}

	for _, channelID := range br.ChannelIDs {
		_, errHangup := h.channelHandler.HangingUp(ctx, channelID, ari.ChannelCauseNormalClearing)
		if errHangup != nil {
			log.WithFields(
				logrus.Fields{
					"bridge_id":  br.ID,
					"channel_id": channelID,
				}).Warningf("Could not hangup the channel. err: %v", err)
		}
	}

	// destroy the confbridge bridge
	if errDestroy := h.bridgeHandler.Destroy(ctx, br.ID); errDestroy != nil {
		log.Errorf("Could not delete confbridge bridge. err: %v", errDestroy)
	}

	return nil
}
