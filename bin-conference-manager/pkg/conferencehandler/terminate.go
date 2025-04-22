package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conference-manager/models/conference"
)

// Terminating is terminating the conference
func (h *conferenceHandler) Terminating(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Terminating",
		"conference_id": id,
	})

	// get conference
	cf, err := h.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	log.WithField("conference", cf).Debug("Found conference info.")

	// if the conference is already terminated or stopping, just return at here
	if cf.Status == conference.StatusTerminated || cf.Status == conference.StatusTerminating {
		log.Infof("The conference is already terminated or terminating. status: %s", cf.Status)
		return cf, nil
	}
	log.Debug("Terminating the conference.")

	res, err := h.UpdateStatus(ctx, id, conference.StatusTerminating)
	if err != nil {
		log.Errorf("Could not update the status for conference terminating. err: %v", err)
		return nil, errors.Wrap(err, "Could not update the status for conference terminating.")
	}

	// kick out all conferencecalls
	for _, confcallID := range cf.ConferencecallIDs {
		_, _ = h.reqHandler.ConferenceV1ConferencecallKick(ctx, confcallID)
	}

	return res, nil
}

// Destroy is terminate the conference without any condition check.
// So, this function must be called in the last step except terminate the conference in forcedly.
func (h *conferenceHandler) Destroy(ctx context.Context, cf *conference.Conference) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Destroy",
		"conference_id": cf.ID,
	})
	log.WithField("conference", cf).Debug("Destroying the conference.")

	// delete confbridge
	tmp, err := h.reqHandler.CallV1ConfbridgeDelete(ctx, cf.ConfbridgeID)
	if err != nil {
		log.WithField("confbridge_id", cf.ConfbridgeID).Errorf("Could not delete the confbridge. But keep moving on. err: %v", err)
	}
	log.WithField("confbridge", tmp).Debug("Deleted the confbridge.")

	// update conference status to terminated
	if err := h.db.ConferenceEnd(ctx, cf.ID); err != nil {
		log.Errorf("Could not terminate the conference. err: %v", err)
		return nil, err
	}
	promConferenceCloseTotal.WithLabelValues(string(cf.Type)).Inc()

	// notify conference deleted event
	res, err := h.Get(ctx, cf.ID)
	if err != nil {
		log.Errorf("Could not get updated conference info. err: %v", err)
		return nil, errors.Wrap(err, "Could not updated conference info.")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conference.EventTypeConferenceDeleted, res)

	return res, nil
}
