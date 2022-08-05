package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// Join handler call's conference join request.
func (h *conferenceHandler) Join(ctx context.Context, conferenceID uuid.UUID, referenceType conferencecall.ReferenceType, referenceID uuid.UUID) (*conferencecall.Conferencecall, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"func":           "Join",
			"conference_id":  conferenceID,
			"reference_type": referenceType,
			"reference_id":   referenceID,
		})
	log.Info("Starting to join the call to the conference.")

	// get conference
	cf, err := h.Get(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return nil, err
	}

	// create a new conferencecall
	res, err := h.conferencecallHandler.Create(ctx, cf.CustomerID, cf.ID, referenceType, referenceID)
	if err != nil {
		log.Errorf("Could not create conferencecall. err: %v", err)
		return nil, err
	}
	log.WithField("conferencecall", res).Debugf("Created a new conferencecall. conferencecall_id: %s", res.ID)

	// put the call into the confbridge
	if err := h.reqHandler.CMV1ConfbridgeCallAdd(ctx, cf.ConfbridgeID, referenceID); err != nil {
		log.Errorf("Could not put the call into the confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}

// JoinedConfbridge handles call's joined confbridge notification
func (h *conferenceHandler) JoinedConfbridge(ctx context.Context, conferenceID, callID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "JoinedConfbridge",
			"conference_id": conferenceID,
			"call_id":       callID,
		})
	log.Info("The call has joined to the confbridge.")

	// update conferencecall
	cc, err := h.conferencecallHandler.UpdateStatusJoinedByReferenceID(ctx, callID)
	if err != nil {
		log.Errorf("Could not update the conferencecall status to the joined. err: %v", err)
		return err
	}

	// add the call to the conference.
	if errAdd := h.db.ConferenceAddConferencecallID(ctx, conferenceID, cc.ID); errAdd != nil {
		log.Errorf("Could not add the conferencecall to the conference. Kicking out the call from the conference. err: %v", errAdd)
		_ = h.reqHandler.CMV1ConfbridgeCallKick(ctx, conferenceID, callID)
		return errAdd
	}

	// send conference notification
	tmpCf, err := h.Get(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get updated conference info. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, tmpCf.CustomerID, conference.EventTypeConferenceUpdated, tmpCf)

	return nil
}
