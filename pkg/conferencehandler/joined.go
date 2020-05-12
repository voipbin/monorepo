package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

func (h *conferenceHandler) Joined(id, callID uuid.UUID) error {
	ctx := context.Background()

	// add the call to conference
	if err := h.db.ConferenceAddCallID(ctx, id, callID); err != nil {
		// we don't kick out the joined call at here.
		// just write log.
		log.WithFields(
			log.Fields{
				"conference": id.String(),
				"call":       callID.String(),
			}).Errorf("Could not add the callid into the conference. err: %v", err)
	}
	return nil
}
