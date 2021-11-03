package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// leaved handles event the channel has left from the bridge
// when the channel has left from the conference bridge, this func will be fired.
func (h *conferenceHandler) Leaved(ctx context.Context, id uuid.UUID, callID uuid.UUID) error {

	log := logrus.WithFields(logrus.Fields{
		"conference_id": id,
		"call_id":       callID,
	})

	if errRemove := h.db.ConferenceRemoveCallID(ctx, id, callID); errRemove != nil {
		log.Errorf("Could not remove the callID from the conference. err: %v", errRemove)
		return errRemove
	}

	// get conference info
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return err
	}

	switch cf.Type {
	case conference.TypeConnect:
		return h.leavedTypeConnect(ctx, cf, callID)

	case conference.TypeConference:
		return h.leavedTypeConference(ctx, cf, callID)

	default:
		log.Errorf("Could not find correct event handler.")
		return fmt.Errorf("could not find connrect event handler")
	}
}

// leavedTypeConnect
func (h *conferenceHandler) leavedTypeConnect(ctx context.Context, cf *conference.Conference, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"conference_id": cf.ID,
		"call_id":       callID,
	})

	if len(cf.CallIDs) <= 0 {
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
func (h *conferenceHandler) leavedTypeConference(ctx context.Context, cf *conference.Conference, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "leavedTypeConference",
		"conference_id": cf.ID,
		"call_id":       callID,
	})

	if cf.Status != conference.StatusTerminating {
		// nothing to do here.
		return nil
	}

	if len(cf.CallIDs) > 0 {
		// we need to wait until all the call has gone
		return nil
	}

	if err := h.Destroy(ctx, cf); err != nil {
		log.Errorf("Could not destory the conference. err: %v", err)
		return err
	}

	return nil
}
