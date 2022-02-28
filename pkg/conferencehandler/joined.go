package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// JoinedConfbridge handles call's joined confbridge notification
func (h *conferenceHandler) JoinedConfbridge(ctx context.Context, confbridgeID, callID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "JoinedConfbridge",
			"confbridge_id": confbridgeID,
			"call_id":       callID,
		})
	log.Info("The call has joined to the confbridge.")

	// get conference
	cf, err := h.db.ConferenceGetByConfbridgeID(ctx, confbridgeID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return err
	}
	log = log.WithField("conference_id", cf.ID)
	log.WithField("conference", cf).Debugf("Found conference info. conference_id: %s", cf.ID)

	// add the call to the conference.
	if errAdd := h.db.ConferenceAddCallID(ctx, cf.ID, callID); errAdd != nil {
		log.Errorf("Could not add the call to the conference. Kicking out the call from the conference. err: %v", errAdd)
		_ = h.reqHandler.CMV1ConfbridgeCallKick(ctx, cf.ID, callID)
		return errAdd
	}

	// send conference notification
	tmpCf, err := h.db.ConferenceGet(ctx, cf.ID)
	if err != nil {
		log.Errorf("Could not get updated conference info. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, tmpCf.CustomerID, conference.EventTypeConferenceUpdated, tmpCf)

	return nil
}
