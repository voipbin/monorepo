package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

// Terminate is final task to terminating the conference
func (h *conferenceHandler) Terminate(id uuid.UUID, reason string) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"conference": id,
			"reason":     reason,
		},
	)

	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		return err
	}

	// if the conference is already terminated or stopping, just return at here
	if cf.Status == conference.StatusTerminated || cf.Status == conference.StatusTerminating {
		logrus.WithFields(
			logrus.Fields{
				"conference": id.String(),
			}).Infof("The conference is already terminated or being terminated. status: %s", cf.Status)

		return nil
	}
	log.Debug("Terminating the conference.")

	// set the status to stopping
	if err := h.db.ConferenceSetStatus(ctx, id, conference.StatusTerminating); err != nil {
		log.WithFields(
			logrus.Fields{
				"conference": id.String(),
			}).Warnf("Could not update the status for conference stopping. err: %v", err)
		return err
	}

	// loop the bridge
	for _, bridgeID := range cf.BridgeIDs {

		// get bridge
		bridge, err := h.db.BridgeGet(ctx, bridgeID)
		if err != nil {
			log.WithFields(
				logrus.Fields{
					"conference": id.String(),
					"bridge":     bridgeID,
				}).Errorf("Could not get bridge.")
			continue
		}

		// we don't hangup the channels in the bridge here.
		// because if the bridge has deleted, all the channels will be left from the bridge,
		// and then the ChannelLeftBridge event will be published.
		// then the ChannelLeftBridge event handler handles the channels.
		// the ChannelLeftBridge will hangup the channel or do the next actions depends on the
		// each channels

		// destroy the bridge
		if err := h.reqHandler.AstBridgeDelete(bridge.AsteriskID, bridge.ID); err != nil {
			log.WithFields(
				logrus.Fields{
					"conference": id.String(),
					"bridge":     bridge.ID,
				}).Errorf("could not delete the bridge. err: %v", err)
			continue
		}
	}

	// update conference status to terminated
	if err := h.db.ConferenceEnd(ctx, id); err != nil {
		log.WithFields(
			logrus.Fields{
				"conference": id.String(),
			}).Errorf("Could not terminate the conference. err: %v", err)
		return err
	}
	promConferenceCloseTotal.WithLabelValues(string(cf.Type)).Inc()

	return nil
}

// hangupAllChannelsInBridge hangs up the all channels in the bridge
func (h *conferenceHandler) hangupAllChannelsInBridge(bridge *bridge.Bridge) {
	logrus.WithFields(
		logrus.Fields{
			"asterisk": bridge.AsteriskID,
			"bridge":   bridge.ID,
			"channels": bridge.ChannelIDs,
		}).Debug("Hanging up all channels in the bridge.")

	// destroy all the channels in the bridge
	for _, channelID := range bridge.ChannelIDs {
		if err := h.reqHandler.AstChannelHangup(bridge.AsteriskID, channelID, ari.ChannelCauseNormalClearing); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"asterisk": bridge.AsteriskID,
					"bridge":   bridge.ID,
					"channel":  channelID,
				}).Warningf("Could not hangup the channel. err: %v", err)
		}
	}
}

// removeAllChannelsInBridge remove the all channels in the bridge
func (h *conferenceHandler) removeAllChannelsInBridge(bridge *bridge.Bridge) {
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
