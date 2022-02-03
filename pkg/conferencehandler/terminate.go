package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
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
	log.WithField("conference", cf).Debug("Found conference info.")

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

	// remove flow
	log.WithField("flow_id", cf.FlowID).Debug("Deleting the flow.")
	if err := h.reqHandler.FMV1FlowDelete(ctx, cf.FlowID); err != nil {
		log.WithField("flow_id", cf.FlowID).Errorf("Could not delete the conference. But keep moving on. err: %v", err)
	}

	// kick out the all calls from the conference.
	log.Debugf("Kicking out all calls from the conference. call_count: %d", len(cf.CallIDs))
	for _, callID := range cf.CallIDs {
		log.Debugf("Kicking out the call. call_id: %v", callID.String())
		if errHangup := h.reqHandler.CMV1ConfbridgeCallKick(ctx, cf.ConfbridgeID, callID); errHangup != nil {
			log.WithField("call_id", callID).Errorf("Could not kicking out the call. err: %v", errHangup)
		}
	}

	if len(cf.CallIDs) == 0 {
		if err := h.Destroy(ctx, cf); err != nil {
			log.Errorf("Could not destroy the conference. err: %v", err)
			return err
		}
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
	if err := h.reqHandler.CMV1ConfbridgeDelete(ctx, cf.ConfbridgeID); err != nil {
		log.WithField("confbridge_id", cf.ConfbridgeID).Errorf("Could not delete the confbridge. But keep moving on. err: %v", err)
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
	h.notifyHandler.PublishWebhookEvent(ctx, tmpConf.CustomerID, conference.EventTypeConferenceDeleted, tmpConf)

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
