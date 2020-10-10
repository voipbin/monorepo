package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/bridge"
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
		log.Infof("The conference is already terminated or being terminated. status: %s", cf.Status)

		return nil
	}
	log.Debug("Terminating the conference.")

	// set the status to stopping
	if err := h.db.ConferenceSetStatus(ctx, id, conference.StatusTerminating); err != nil {
		log.Warnf("Could not update the status for conference stopping. err: %v", err)
		return err
	}

	// hangup all channels from the conference bridge
	br, err := h.db.BridgeGet(ctx, cf.BridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. bridge: %s, err: %v", cf.BridgeID, err)
		return err
	}
	h.hangupAllChannelsInBridge(br)

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
