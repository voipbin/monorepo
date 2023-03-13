package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// Terminate is terminating the conference
func (h *conferenceHandler) Terminate(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Terminate",
		"conference_id": id,
	})

	// get conference
	cf, err := h.Get(ctx, id)
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
	_, err = h.reqHandler.FlowV1FlowDelete(ctx, cf.FlowID)
	if err != nil {
		log.WithField("flow_id", cf.FlowID).Errorf("Could not delete the conference. But keep moving on. err: %v", err)
	}

	// delete the conference
	if err := h.Destroy(ctx, cf); err != nil {
		log.Errorf("Could not destroy the conference. err: %v", err)
		return err
	}

	return nil
}

// Destroy is terminate the conference without any condition check.
// So, this function must be called in the last step except terminate the conference in forcedly.
func (h *conferenceHandler) Destroy(ctx context.Context, cf *conference.Conference) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Destroy",
		"conference_id": cf.ID,
	})
	log.WithField("conference", cf).Debug("Destroying the conference.")

	// delete confbridge
	if err := h.reqHandler.CallV1ConfbridgeDelete(ctx, cf.ConfbridgeID); err != nil {
		log.WithField("confbridge_id", cf.ConfbridgeID).Errorf("Could not delete the confbridge. But keep moving on. err: %v", err)
	}

	// update conference status to terminated
	if err := h.db.ConferenceEnd(ctx, cf.ID); err != nil {
		log.Errorf("Could not terminate the conference. err: %v", err)
		return err
	}
	promConferenceCloseTotal.WithLabelValues(string(cf.Type)).Inc()

	// notify conference deleted event
	tmpConf, err := h.Get(ctx, cf.ID)
	if err != nil {
		log.Errorf("Could not get updated conference info. err: %v", err)
		return nil
	}
	h.notifyHandler.PublishWebhookEvent(ctx, tmpConf.CustomerID, conference.EventTypeConferenceDeleted, tmpConf)

	return nil
}
