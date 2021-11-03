package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Join handler call's conference join request.
func (h *conferenceHandler) Join(ctx context.Context, conferenceID, callID uuid.UUID) error {

	log := logrus.WithFields(
		logrus.Fields{
			"conference": conferenceID,
			"call":       callID,
		})
	log.Info("Starting to join the call to the conference.")

	// get conference
	cf, err := h.db.ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return err
	}

	// put the call into the confbridge
	if err := h.reqHandler.CMConfbridgesIDCallsIDPost(cf.ConfbridgeID, callID); err != nil {
		log.Errorf("Could not put the call into the confbridge. err: %v", err)
		return err
	}

	return nil
}
