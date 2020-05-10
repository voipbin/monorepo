package conferhandler

import (
	"context"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
)

// Terminate is final task to terminating the conference
func (h *conferHandler) Terminate(id uuid.UUID) error {
	ctx := context.Background()

	// update conference status to terminated
	if err := h.db.ConferenceEnd(ctx, id); err != nil {
		log.WithFields(
			log.Fields{
				"conference": id.String(),
			}).Errorf("Could not terminate the conference. err: %v", err)
		return err
	}

	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		return err
	}

	// loop the bridge
	for _, bridgeID := range cf.BridgeIDs {

		// get bridge
		bridge, err := h.db.BridgeGet(ctx, bridgeID)
		if err != nil {
			log.WithFields(
				log.Fields{
					"conference": id.String(),
					"bridge":     bridgeID,
				}).Errorf("Could not get bridge.")
			continue
		}

		// hangup all channels
		h.hangupAllChannelsInBridge(bridge)
	}

	return nil
}

// hangupAllChannelsInBridge hangs up the all channels in the bridge
func (h *conferHandler) hangupAllChannelsInBridge(bridge *bridge.Bridge) {
	log.WithFields(
		log.Fields{
			"asterisk": bridge.AsteriskID,
			"bridge":   bridge.ID,
			"channels": bridge.ChannelIDs,
		}).Debug("Hanging up all channels in the bridge.")

	// destroy all the channels in the bridge
	for _, channelID := range bridge.ChannelIDs {
		if err := h.reqHandler.AstChannelHangup(bridge.AsteriskID, channelID, ari.ChannelCauseNormalClearing); err != nil {
			log.WithFields(
				log.Fields{
					"asterisk": bridge.AsteriskID,
					"bridge":   bridge.ID,
					"channel":  channelID,
				}).Warningf("Could not hangup the channel. err: %v", err)
		}
	}
}
