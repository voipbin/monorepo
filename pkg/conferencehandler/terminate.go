package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/notifyhandler"
)

// Terminate is terminating the conference
func (h *conferenceHandler) Terminate(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"conference_id": id,
		},
	)

	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		return err
	}
	log.Debugf("Founc conference info. conference: %v", cf)

	// if the conference is already terminated or stopping, just return at here
	if cf.Status == conference.StatusTerminated || cf.Status == conference.StatusTerminating {
		log.Infof("The conference is already terminated or terminating. status: %s", cf.Status)
		return nil
	}
	log.Debug("Terminating the conference.")

	// set the status to stopping
	if err := h.db.ConferenceSetStatus(ctx, id, conference.StatusTerminating); err != nil {
		log.Errorf("Could not update the status for conference stopping. err: %v", err)
		return err
	}

	// terminate confbridge
	// this will kick out all the calls in the conference(confbridge).
	if err := h.reqHandler.CMConfbridgesIDDelete(cf.ConfbridgeID); err != nil {
		log.Errorf("Could not delete the confbridge. err: %v", err)
		return err
	}

	return nil
}

// Destroy is terminate the conference without any condition check.
// So, this function must be called in the last step except terminate the conference in forcedly.
func (h *conferenceHandler) Destroy(ctx context.Context, cf *conference.Conference) error {
	log := logrus.WithFields(
		logrus.Fields{
			"conference_id": cf.ID,
			"func":          "Destroy",
		},
	)
	log.WithField("conference", cf).Debug("Destroying the conference.")

	// delete confbridge
	if err := h.reqHandler.CMConfbridgesIDDelete(cf.ConfbridgeID); err != nil {
		log.Errorf("Could not delete the confbridge. confbridge_id: %v, err: %v", cf.ConfbridgeID, err)
		return err
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
// func (h *conferenceHandler) hangupAllChannelsInBridge(bridge *bridge.Bridge) {
// logrus.WithFields(
// 	logrus.Fields{
// 		"asterisk": bridge.AsteriskID,
// 		"bridge":   bridge.ID,
// 		"channels": bridge.ChannelIDs,
// 	}).Debug("Hanging up all channels in the bridge.")

// // destroy all the channels in the bridge
// for _, channelID := range bridge.ChannelIDs {
// 	if err := h.reqHandler.AstChannelHangup(bridge.AsteriskID, channelID, ari.ChannelCauseNormalClearing); err != nil {
// 		logrus.WithFields(
// 			logrus.Fields{
// 				"asterisk": bridge.AsteriskID,
// 				"bridge":   bridge.ID,
// 				"channel":  channelID,
// 			}).Warningf("Could not hangup the channel. err: %v", err)
// 	}
// }
// }
