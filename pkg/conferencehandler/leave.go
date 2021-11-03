package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Leave outs the call from the conference
func (h *conferenceHandler) Leave(ctx context.Context, id, callID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"conference": id.String(),
			"call":       callID.String(),
		})
	log.Debugf("Leaving the call from the conference.")

	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return err
	}

	// check the call does exist
	if !isCallExist(cf.CallIDs, callID) {
		log.Errorf("The call does not exist in the conference.")
		return fmt.Errorf("call does not exist")
	}

	// send the kick request
	if err := h.reqHandler.CMConfbridgesIDCallsIDDelete(id, callID); err != nil {
		log.Errorf("Could not kick the call from the conference. err: %v", err)
		return err
	}

	return nil
}

// isCallExist returns true if the call is exist in the calls.
func isCallExist(calls []uuid.UUID, callID uuid.UUID) bool {
	for _, id := range calls {
		if id == callID {
			return true
		}
	}

	return false
}
