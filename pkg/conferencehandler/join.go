package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

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
	if err := h.reqHandler.CallV1ConfbridgeCallAdd(ctx, cf.ConfbridgeID, referenceID); err != nil {
		log.Errorf("Could not put the call into the confbridge. err: %v", err)
		return nil, err
	}

	return res, nil
}
