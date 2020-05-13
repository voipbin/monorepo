package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

// Leaved handle
func (h *conferenceHandler) Leaved(id, callID uuid.UUID) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"conference": id.String(),
			"Call":       callID.String(),
		})
	log.Debug("The call has leaved from the conference.")

	if err := h.db.ConferenceRemoveCallID(ctx, id, callID); err != nil {
		log.Errorf("Could not remove the call id from the conference. err: %v", err)
		return err
	}

	// evaluate the conference is terminatable
	if h.isTerminatable(ctx, id) == true {
		log.Info("This conference is ended. Terminating the conference.")
		return h.Terminate(id)
	}

	return nil
}

// isTerminatable returns true if the given conference is terminatable
// return false if it gets error
func (h *conferenceHandler) isTerminatable(ctx context.Context, id uuid.UUID) bool {
	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		// the call has removed already.
		// we don't need to advertise the err at here.
		// just write log
		log.WithFields(
			log.Fields{
				"conference": id.String(),
			}).Warningf("Could not get conference after remove the call. err: %v", err)
		return false
	}

	// check there's more calls or not
	callCnt := len(cf.CallIDs)

	switch cf.Type {
	case conference.TypeEcho:
		if callCnt <= 0 {
			return true
		}
	default:
		return true
	}

	return false
}
