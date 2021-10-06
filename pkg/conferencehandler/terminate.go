package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// Terminate is terminating the conference
func (h *conferenceHandler) Terminate(id uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"conference": id,
		},
	)

	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		return err
	}

	// if the conference is already terminated or stopping, just return at here
	if cf.Status == conference.StatusTerminated || cf.Status == conference.StatusTerminating {
		log.Infof("The conference is already terminated or terminating. status: %s", cf.Status)
		return nil
	}
	log.Debug("Terminating the conference.")

	// set the status to stopping
	if err := h.db.ConferenceSetStatus(ctx, id, conference.StatusTerminating); err != nil {
		log.Warnf("Could not update the status for conference stopping. err: %v", err)
		return err
	}

	// check remains calls in the conference
	if len(cf.CallIDs) > 0 {
		// hangup all channels from the conference bridge
		br, err := h.db.BridgeGet(ctx, cf.BridgeID)
		if err != nil {
			log.Errorf("Could not get bridge info. bridge: %s, err: %v", cf.BridgeID, err)
			return err
		}
		log.Infof("The conference is terminating. Hangup all channels in conference bridge. conference: %s", cf.ID)

		h.hangupAllChannelsInBridge(br)

		// we are kicking out the all calls from the conference.
		// but we don't cleaning the list of the calls in the conference here.
		// that will be done when the channel left from the call bridge.

		return nil
	}

	// no calls left. destroy the conference
	return h.Destroy(id)
}

// Destroy is terminate the conference without any condition check.
// So, this function must be called in the last step except terminate the conference in forcedly.
func (h *conferenceHandler) Destroy(id uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"conference_id": id,
			"func":          "Destroy",
		},
	)

	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return err
	}
	log.WithField("conference", cf).Debug("Destroying the conference.")

	// get bridge info
	br, err := h.db.BridgeGet(ctx, cf.BridgeID)
	if err == nil && br != nil {
		log.WithField("bridge", br).Debug("Found conference bridge info.")

		if len(br.ChannelIDs) > 0 {
			log.Errorf("There are channels in the conference bridge. We can't destroy the conference now.")
			return fmt.Errorf("channels are in the conference bridge")
		}

		// delete the conference bridge
		if err := h.reqHandler.AstBridgeDelete(br.AsteriskID, br.ID); err != nil {
			log.Errorf("Could not delete the conference bridge correctly. err: %v", err)
			// this is ok. we don't return the error here
		}
	}

	// update conference status to terminated
	if err := h.db.ConferenceEnd(ctx, cf.ID); err != nil {
		log.Errorf("Could not terminate the conference. err: %v", err)
		return err
	}

	promConferenceCloseTotal.WithLabelValues(string(cf.Type)).Inc()

	// notify conference deleted event
	tmpConf, err := h.db.ConferenceGet(ctx, cf.ID)
	if err != nil {
		log.Errorf("Could not get updated conference info. err: %v", err)
		return nil
	}
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeConferenceDeleted, tmpConf.WebhookURI, tmpConf)

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
