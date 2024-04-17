package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// Leaved handles event the referencecall has left from the confbridge
// func (h *conferenceHandler) Leaved(ctx context.Context, cfID uuid.UUID, ccID uuid.UUID) (*conference.Conference, error) {
func (h *conferenceHandler) RemoveConferencecallID(ctx context.Context, cfID uuid.UUID, ccID uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "RemoveConferencecallID",
		"conference_id":     cfID,
		"conferencecall_id": ccID,
	})

	// remove conferencecall from the conference
	if errRemove := h.db.ConferenceRemoveConferencecallID(ctx, cfID, ccID); errRemove != nil {
		log.Errorf("Could not remove the callID from the conference. err: %v", errRemove)
		return nil, errRemove
	}

	// get updated conference
	res, err := h.Get(ctx, cfID)
	if err != nil {
		log.Errorf("Could not get updated conference. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conference.EventTypeConferenceUpdated, res)

	go func() {
		switch res.Type {

		default:
			log.Debugf("Executing default conference leave handle. conference_type: %s", res.Type)
			if errLeaved := h.removeConferencecallIDTypeConference(ctx, res); errLeaved != nil {
				log.Errorf("Could not complete the process. err: %v", errLeaved)
			}
		}
	}()

	return res, nil
}

// removeConferencecallIDTypeConference
func (h *conferenceHandler) removeConferencecallIDTypeConference(ctx context.Context, cf *conference.Conference) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "removeConferencecallIDTypeConference",
		"conference": cf,
	})

	if cf.Status != conference.StatusTerminating {
		// nothing to do here.
		return nil
	}

	if len(cf.ConferencecallIDs) > 0 {
		// we need to wait until all the call has gone
		return nil
	}

	_, err := h.Destroy(ctx, cf)
	if err != nil {
		log.Errorf("Could not destory the conference. err: %v", err)
		return err
	}

	return nil
}
