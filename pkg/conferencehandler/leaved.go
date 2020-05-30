package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

// Leaved handle
func (h *conferenceHandler) Leaved(id, callID uuid.UUID) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"conference": id.String(),
			"Call":       callID.String(),
		})
	log.Debug("The call has leaved from the conference.")

	if err := h.db.ConferenceRemoveCallID(ctx, id, callID); err != nil {
		log.Errorf("Could not remove the call id from the conference. err: %v", err)
		return err
	}

	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. conference: %s, err: %v", id, err)
		return err
	}
	promConferenceLeaveTotal.WithLabelValues(string(cf.Type)).Inc()

	// evaluate the conference is terminatable
	if h.isTerminatable(ctx, id) == true {
		log.Info("This conference is ended. Terminating the conference.")
		return h.Terminate(id)
	}

	return nil
}

// leaved handle
func (h *conferenceHandler) leaved(cn *channel.Channel, br *bridge.Bridge) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"conference":      br.ConferenceID,
			"conference_type": br.ConferenceType,
			"conference_join": br.ConferenceJoin,
			"channel":         cn.ID,
			"bridge":          br.ID,
		},
	)

	id := br.ConferenceID

	// get call info
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	// add the call id to the log
	log = log.WithFields(
		logrus.Fields{
			"call": c.ID,
		},
	)

	// set empty conference id
	if err := h.db.CallSetConferenceID(ctx, c.ID, uuid.Nil); err != nil {
		log.Errorf("Could not reset the conference for a call. err: %v", err)
		return err
	}
	log.Debug("The call has been leaved from the conference.")

	// remove the call from the conference
	if err := h.db.ConferenceRemoveCallID(ctx, id, c.ID); err != nil {
		log.Errorf("Could not remove the call id from the conference. err: %v", err)
		return err
	}
	promConferenceLeaveTotal.WithLabelValues(string(br.ConferenceType)).Inc()

	// send a call action next after all has done
	if err := h.reqHandler.CallCallActionNext(c.ID); err != nil {
		log.Errorf("Could not send the call action next request.")
		return err
	}

	// evaluate the conference is terminatable
	if h.isTerminatable(ctx, id) == true {
		log.Info("This conference is ended. Terminating the conference.")
		return h.Terminate(id)
	}

	return nil
}

// isTerminatable returns true if the given conference is terminatable
// return false if it gets error
func (h *conferenceHandler) isTerminatable(ctx context.Context, id uuid.UUID) bool {
	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		// the call has removed already.
		// we don't need to advertise the err at here.
		// just write log
		log.WithFields(
			log.Fields{
				"conference": id.String(),
			}).Warningf("Could not get conference after remove the call. err: %v", err)
		return false
	}

	// check there's more calls or not
	callCnt := len(cf.CallIDs)

	switch cf.Type {
	case conference.TypeEcho:
		if callCnt <= 0 {
			return true
		}
	default:
		return true
	}

	return false
}
