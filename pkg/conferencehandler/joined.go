package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// JoinedConfbridge handles call's joined confbridge notification
func (h *conferenceHandler) JoinedConfbridge(ctx context.Context, conferenceID, callID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "JoinedConfbridge",
			"conference_id": conferenceID,
			"call_id":       callID,
		})
	log.Info("The call has joined to the confbridge.")

	// add the call to the conference.
	if errAdd := h.db.ConferenceAddCallID(ctx, conferenceID, callID); errAdd != nil {
		log.Errorf("Could not add the call to the conference. Kicking out the call from the conference. err: %v", errAdd)
		_ = h.reqHandler.CMV1ConfbridgeCallKick(ctx, conferenceID, callID)
		return errAdd
	}

	// send conference notification
	tmpCf, err := h.db.ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get updated conference info. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, tmpCf.CustomerID, conference.EventTypeConferenceUpdated, tmpCf)

	return nil
}
