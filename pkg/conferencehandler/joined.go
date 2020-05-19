package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

func (h *conferenceHandler) Joined(id, callID uuid.UUID) error {
	ctx := context.Background()
	log := log.WithFields(
		log.Fields{
			"conference": id,
			"call":       callID,
		})
	log.Debug("The call has joined into the conference.")

	// add the call to conference
	if err := h.db.ConferenceAddCallID(ctx, id, callID); err != nil {
		// we don't kick out the joined call at here.
		// just write log.
		log.Errorf("Could not add the callid into the conference. conference: %s, call: %s, err: %v", id, callID, err)
	}

	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. conference: %s, err: %v", id, err)
	}
	promConferenceJoinTotal.WithLabelValues(string(cf.Type)).Inc()

	return nil
}
