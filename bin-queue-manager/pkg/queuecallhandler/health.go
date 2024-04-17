package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// HealthCheck checks the given call is still vaild
// and hangup the call if the call is not valid over the default retry count.
func (h *queuecallHandler) HealthCheck(ctx context.Context, id uuid.UUID, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "HealthCheck",
		"queuecall_id": id,
		"retry_count":  retryCount,
	})

	// validate the queuecall.
	qc, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queuecall info. err: %v", err)
		return
	}

	// check the queuecall is still ongoing
	if qc.Status == queuecall.StatusAbandoned || qc.Status == queuecall.StatusDone {
		// the call is already done. no need to check the health anymore.
		log.Debugf("The queuecall is already done. No need to check the health anymore. queuecalll_id: %v", qc.ID)
		return
	}

	// check reference type
	if qc.ReferenceType != queuecall.ReferenceTypeCall {
		log.Debugf("The queuecall reference type is not a call. No need to check the health anymore. queuecall_id: %v", qc)
		return
	}

	if retryCount > defaultHealthCheckMaxRetryCount {
		log.Debugf("Exceeded the max retry count. No need to check the health anymore. queuecall_id: %v", qc.ID)
		res, err := h.kickForce(ctx, id)
		if err != nil {
			log.Errorf("Could not kick force the queuecall. err: %v", err)
			return
		}
		log.WithField("queuecall", res).Debugf("Kick force the queuecall. queuecall_id: %s", res.ID)
		return
	}

	// increase retry count
	newCount := retryCount + 1

	// get reference call info
	c, err := h.reqHandler.CallV1CallGet(ctx, qc.ReferenceID)
	if err != nil {
		log.Errorf("Could not get call info: %v", err)
		// send request again with increased retry count
		_ = h.reqHandler.QueueV1QueuecallHealthCheck(ctx, qc.ID, defaultHealthCheckDelay, newCount)
		return
	}
	log.WithField("call", c).Debugf("Found reference call info.")

	// check call's status
	if c.Status == cmcall.StatusHangup {
		// the reference call is already done. will che
		// send retry count with 3 second delay
		_ = h.reqHandler.QueueV1QueuecallHealthCheck(ctx, qc.ID, defaultHealthCheckDelay, newCount)
		return
	}

	// everyting is fine. reset the retry count to the 0
	_ = h.reqHandler.QueueV1QueuecallHealthCheck(ctx, qc.ID, defaultHealthCheckDelay, 0)
}
