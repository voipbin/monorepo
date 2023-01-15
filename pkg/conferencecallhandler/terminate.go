package conferencecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// Terminate terminates the conferencecall.
func (h *conferencecallHandler) Terminate(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "Terminate",
			"conferencecall_id": id,
		},
	)

	// get conferencecall
	cc, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conferencecall info. err: %v", err)
		return nil, err
	}

	// get conference
	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, cc.ConferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. conference_id: %s, err: %v", cf.ID, err)
		return nil, err
	}

	// get call info
	c, err := h.reqHandler.CallV1CallGet(ctx, cc.ReferenceID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	if !h.isKickable(ctx, cc, cf, c) {
		res, err := h.UpdateStatusLeaved(ctx, id)
		if err != nil {
			log.Errorf("Could not update the status to leaved. err: %v", err)
			return nil, err
		}

		// send remove conferencecall
		tmp, err := h.reqHandler.ConferenceV1ConferenceRemoveConferencecallID(ctx, cf.ID, cc.ID)
		if err != nil {
			log.Errorf("Could not remove the conferencecall id from the conference. err: %v", err)
			return nil, err
		}
		log.WithField("conference", tmp).Debugf("Removed conferencecall from the conference. conference_id: %s, conferencecall_id: %S", cf.ID, cc.ID)

		return res, nil
	}

	// update the conferencecall
	res, err := h.updateStatus(ctx, id, conferencecall.StatusLeaving)
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
func (h *conferencecallHandler) isKickable(ctx context.Context, cc *conferencecall.Conferencecall, cf *conference.Conference, c *cmcall.Call) bool {

	// check the conferencecall's status
	if cc.Status == conferencecall.StatusLeaved || cc.Status == conferencecall.StatusLeaving {
		return false
	}

	// check the call status
	if c.Status != cmcall.StatusProgressing {
		return false
	}

	// check the call's confbridge info
	if c.ConfbridgeID != cf.ConfbridgeID {
		return false
	}

	// check the confernce status
	if cf.Status == conference.StatusTerminated {
		return false
	}

	return true
}

// Terminated handles terminated conferencecall
func (h *conferencecallHandler) Terminated(ctx context.Context, cc *conferencecall.Conferencecall) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "Terminated",
			"conferencecall_id": cc.ID,
		},
	)

	// update status
	res, err := h.UpdateStatusLeaved(ctx, cc.ID)
	if err != nil {
		log.Errorf("Could not update the conferencecall status. err: %v", err)
		return nil, err
	}

	// send request
	cf, err := h.reqHandler.ConferenceV1ConferenceRemoveConferencecallID(ctx, cc.ConferenceID, cc.ID)
	if err != nil {
		log.Errorf("Could not remove the conferencecall id from the conference. err: %v", err)
		return nil, err
	}
	log.WithField("conference", cf).Debugf("Removed conferencecall id from the conference. conference_id: %s", cf.ID)

	return res, nil
}
