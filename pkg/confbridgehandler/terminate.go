package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
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
	log.WithField("confbridge", cb).Debugf("Found confbridge info. confbridge_id: %s", cb.ID)

	if cb.BridgeID != "" {
		log.Debugf("The confbridge has bridge id. Destroying the bridge. bridge_id: %s", cb.BridgeID)
		if errDestroy := h.destroyBridge(ctx, cb.BridgeID); errDestroy != nil {
			// could not destroy the bridge. but we don't return the error here. just write the error log
			log.Errorf("Could not destroy the bridge. err: %v", errDestroy)
		}
	}

	_, err = h.Delete(ctx, cb.ID)
	if err != nil {
		log.Errorf("Could not delete the confbridge. err: %v", err)
		return err
	}

	return nil
}

// destroyBridge hangs up all the channels from the bridge and destroy it.
func (h *confbridgeHandler) destroyBridge(ctx context.Context, bridgeID string) error {
	log := logrus.WithFields(logrus.Fields{
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
			log.WithFields(logrus.Fields{
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
