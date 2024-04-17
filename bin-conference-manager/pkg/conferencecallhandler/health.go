package conferencecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// HealthCheck checks the given conferencecall's status.
// if the status is not valid it removes the conferencecall from the conference.
func (h *conferencecallHandler) HealthCheck(ctx context.Context, id uuid.UUID, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "HealthCheck",
		"conferencecall_id": id,
	})

	// check the retry count
	if retryCount > defaultHealthCheckRetryMax {
		log.Debugf("The conferencecall's healthcheck exceeded max retry count. Terminate the conferencecall. retry_count: %d", retryCount)
		tmp, err := h.Terminate(ctx, id)
		if err != nil {
			log.Errorf("Could not terminate the conferencecall. err: %v", err)
			return
		}
		log.WithField("conferencecall", tmp).Debugf("Terminated conferencecall. conferencecall_id: %s", tmp.ID)
		return
	}

	// get conferencecall
	cc, err := h.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get conferencecall. err: %v", err)
		go func() {
			_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, retryCount+1, defaultHealthCheckDelay)
		}()
		return
	}

	// check max conferencecall duration
	tmTimeout := h.utilHandler.TimeGetCurTimeAdd(-maxConferencecallDuration)
	if cc.TMCreate < tmTimeout {
		log.Debugf("Exceed max conferencecall duration. max_duration: %s", tmTimeout)
		go func() {
			_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, retryCount+1, defaultHealthCheckDelay)
		}()
		return
	}

	// check conferencecall's status
	if cc.Status == conferencecall.StatusLeaved {
		// already done. no need healthcheck anymore.
		return
	}

	// get call info
	c, err := h.reqHandler.CallV1CallGet(ctx, cc.ReferenceID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		go func() {
			_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, retryCount+1, defaultHealthCheckDelay)
		}()
		return
	}

	// check the call status
	if c.Status != cmcall.StatusProgressing {
		log.Debugf("The call has invalid status.")
		go func() {
			_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, retryCount+1, defaultHealthCheckDelay)
		}()
		return
	}

	// get conference info
	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, cc.ConferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		go func() {
			_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, retryCount+1, defaultHealthCheckDelay)
		}()
		return
	}

	if cf.Status != conference.StatusProgressing {
		// conference status is invalid
		go func() {
			_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, retryCount+1, defaultHealthCheckDelay)
		}()
		return
	}

	// we don't check the call and conference's confbridge id
	// because the call-manager updates the call's confbridge id after done the pre-action execution.
	// so it limits the conference's pre-action execution.
	//
	// if cf.ConfbridgeID != c.ConfbridgeID {
	// 	log.Errorf("The call has invalid confbridge info. call_id: %s, confbridge_id: %s", c.ID, c.ConfbridgeID)
	// 	go func() {
	// 		_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, retryCount+1, defaultHealthCheckDelay)
	// 	}()
	// 	return
	// }

	log.Debugf("The call is still going on.")
	go func() {
		_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, 0, defaultHealthCheckDelay)
	}()
}
