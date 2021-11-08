package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// Terminate is terminating the conference
func (h *confbridgeHandler) Terminate(id uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithField("confbridge_id", id)

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
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeConfbridgeDeleted, "", tmpCB)

	return nil
}

// destroyBridge hangs up all the channels from the bridge and destroy it.
func (h *confbridgeHandler) destroyBridge(ctx context.Context, bridgeID string) error {
	log := logrus.WithField("func", "hangupAllChannels")

	if !h.isBridgeExist(ctx, bridgeID) {
		return nil
	}

	// hang up all the channels in the bridge
	br, err := h.db.BridgeGet(ctx, bridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return err
	}

	for _, channelID := range br.ChannelIDs {
		if err := h.reqHandler.AstChannelHangup(br.AsteriskID, channelID, ari.ChannelCauseNormalClearing); err != nil {
			log.WithFields(
				logrus.Fields{
					"asterisk": br.AsteriskID,
					"bridge":   br.ID,
					"channel":  channelID,
				}).Warningf("Could not hangup the channel. err: %v", err)
		}
	}

	// destroy the confbridge bridge
	if errBridgeDel := h.reqHandler.AstBridgeDelete(br.AsteriskID, br.ID); errBridgeDel != nil {
		log.Errorf("Could not delete confbridge bridge. err: %v", errBridgeDel)
	}

	return nil
}
