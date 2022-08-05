package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// Leaved handles event the referencecall has left from the confbridge
func (h *conferenceHandler) Leaved(ctx context.Context, cf *conference.Conference, referenceID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Leaved",
		"conference_id": cf.ID,
		"reference_id":  referenceID,
	})

	// get conferencecall
	cc, err := h.conferencecallHandler.GetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get conferencecall info. err: %v", err)
		return err
	}

	// update conferencecall status
	_, err = h.conferencecallHandler.UpdateStatusLeaved(ctx, cc.ID)
	if err != nil {
		log.Errorf("Could not update the conferencecall's status. err: %v", err)
		return err
	}

	// remove call from the conference
	if errRemove := h.db.ConferenceRemoveConferencecallID(ctx, cf.ID, cc.ID); errRemove != nil {
		log.Errorf("Could not remove the callID from the conference. err: %v", errRemove)
		return errRemove
	}

	// get updated conference
	tmp, err := h.Get(ctx, cf.ID)
	if err != nil {
		log.Errorf("Could not get updated conference. err: %v", err)
		return err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, tmp.CustomerID, conference.EventTypeConferenceUpdated, tmp)

	switch cf.Type {
	case conference.TypeConnect:
		return h.leavedTypeConnect(ctx, tmp)

	case conference.TypeConference:
		return h.leavedTypeConference(ctx, tmp)

	default:
		log.Errorf("Could not find correct event handler.")
		return fmt.Errorf("could not find connrect event handler")
	}
}

// leavedTypeConnect
func (h *conferenceHandler) leavedTypeConnect(ctx context.Context, cf *conference.Conference) error {
	log := logrus.WithFields(logrus.Fields{
		"conference_id": cf.ID,
	})

	if len(cf.ConferencecallIDs) <= 0 {
		if err := h.Destroy(ctx, cf); err != nil {
			log.Errorf("Could not destroy the conference. err: %v", err)
			return err
		}
	} else {
		if err := h.Terminate(ctx, cf.ID); err != nil {
			log.Errorf("Could not terminate the conference. err: %v", err)
			return err
		}
	}

	return nil
}

// leavedTypeConference
func (h *conferenceHandler) leavedTypeConference(ctx context.Context, cf *conference.Conference) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "leavedTypeConference",
		"conference_id": cf.ID,
	})

	if cf.Status != conference.StatusTerminating {
		// nothing to do here.
		return nil
	}

	if len(cf.ConferencecallIDs) > 0 {
		// we need to wait until all the call has gone
		return nil
	}

	if err := h.Destroy(ctx, cf); err != nil {
		log.Errorf("Could not destory the conference. err: %v", err)
		return err
	}

	return nil
}
