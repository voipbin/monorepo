package conferencecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conference-manager/models/conferencecall"
)

// Terminate terminates the conferencecall.
func (h *conferencecallHandler) Terminate(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "Terminate",
		"conferencecall_id": id,
	})

	// get conferencecall
	cc, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conferencecall info. err: %v", err)
		return nil, err
	}
	log = log.WithField("reference_id", cc.ReferenceID)

	// get conference
	cf, err := h.conferenceHandler.Get(ctx, cc.ConferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. conference_id: %s, err: %v", cf.ID, err)
		return nil, err
	}
	log = log.WithField("conference_id", cf.ID)

	if !h.isKickable(ctx, cc) {
		// it's not kickable. nothing to do.
		log.WithField("conferncecall", cc).Debugf("The conferencecall is not kickable. Nothing to do. conferencecall_id: %s", cc.ID)
		return cc, nil
	}

	// update the conferencecall
	res, err := h.updateStatusLeaving(ctx, id)
	if err != nil {
		log.Errorf("Could not update the status correctly. err: %v", err)
		return nil, err
	}

	// send the kick request
	if err := h.reqHandler.CallV1ConfbridgeCallKick(ctx, cf.ConfbridgeID, cc.ReferenceID); err != nil {
		log.Errorf("Could not kick the call from the conference. err: %v", err)
		return nil, err
	}

	return res, nil
}

// isKickable returns true if the given conferencecall is kickable
func (h *conferencecallHandler) isKickable(ctx context.Context, cc *conferencecall.Conferencecall) bool {

	// check the conferencecall's status
	if cc.Status == conferencecall.StatusLeaved || cc.Status == conferencecall.StatusLeaving {
		return false
	}

	return true
}

// Terminated handles terminated conferencecall
func (h *conferencecallHandler) Terminated(ctx context.Context, cc *conferencecall.Conferencecall) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Terminated",
		"conferencecall": cc,
	})

	// update status
	res, err := h.updateStatusLeaved(ctx, cc.ID)
	if err != nil {
		log.Errorf("Could not update the conferencecall status. err: %v", err)
		return nil, err
	}

	// send request
	cf, err := h.conferenceHandler.RemoveConferencecallID(ctx, cc.ConferenceID, cc.ID)
	if err != nil {
		log.Errorf("Could not remove the conferencecall id from the conference. err: %v", err)
		return nil, err
	}
	log.WithField("conference", cf).Debugf("Removed conferencecall id from the conference. conference_id: %s", cf.ID)

	return res, nil
}
