package conferencecallhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// Joined handles joined conferencecall
func (h *conferencecallHandler) Joined(ctx context.Context, cc *conferencecall.Conferencecall) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "Joined",
		"conferencecall_id": cc.ID,
		"conference_id":     cc.ConferenceID,
		"reference_type":    cc.ReferenceType,
		"reference_id":      cc.ReferenceID,
	})

	// update status
	res, err := h.updateStatusJoined(ctx, cc.ID)
	if err != nil {
		log.Errorf("Could not update the conferencecall status. err: %v", err)
		return nil, err
	}

	// add conferencecall to the conference
	cf, err := h.conferenceHandler.AddConferencecallID(ctx, cc.ConferenceID, cc.ID)
	if err != nil {
		log.Errorf("Could not remove the conferencecall id from the conference. err: %v", err)
		return nil, err
	}
	log.WithField("conference", cf).Debugf("Added conferencecall id to the conference. conference_id: %s", cf.ID)

	return res, nil
}
