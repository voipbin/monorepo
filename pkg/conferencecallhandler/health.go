package conferencecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// HealthCheck checks the given conferencecall's status.
// if the status is not valid it removes the conferencecall from the conference.
func (h *conferencecallHandler) HealthCheck(ctx context.Context, id uuid.UUID, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "HealthCheck",
		"conferencecall_id": id,
	})

	// get conferencecall
	cc, err := h.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get conferencecall. err: %v", err)
		return
	}

	if cc.Status == conferencecall.StatusLeaved || cc.Status == conferencecall.StatusLeaving {
		// nothing to do anymore
		return
	}

	// get call info
	c, err := h.reqHandler.CallV1CallGet(ctx, cc.ID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return
	}

	// check the call status
	if c.Status == cmcall.StatusProgressing {
		// send the request with 5 seconds delay
		log.Debugf("The call is still ongoing.")
	}

	// the conferencecall already gone
	// send leaved request

	if retryCount < defaultHealthCheckRetryMax {
		// send health check
	}
}
