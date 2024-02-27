package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
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

	// check the queuecall is still valid
	if qc.Status == queuecall.StatusAbandoned || qc.Status == queuecall.StatusDone {
		// the call is already done. no need to check the health anymore.
		log.Debugf("The queuecall is already done. no need to check the health anymore. queuecalll_id: %v", qc.ID)
		return
	}

	// check reference type
	if qc.ReferenceType == queuecall.ReferenceTypeCall {
		log.Debugf("The queuecall reference type is not call. No need to check the health anymore. queuecall_id: %v", qc)
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

	// get call info
	c, err := h.reqHandler.CallV1CallGet(ctx, qc.ReferenceID)
	if err != nil {
		log.Errorf("Could not get call info: %v", err)
		// send retry count with 3 second delay
	}
	log.WithField("call", c).Debugf("Found call info.")

	// // check call's status
	// if c.Status == cmcall.StatusHangup {
	// 	// the reference call is already done. will che
	// 	// send retry count with 3 second delay
	// }

	// // validate call's channel.
	// cn, err := h.channelHandler.Get(ctx, qc.ChannelID)
	// if err != nil {
	// 	log.Errorf("Could not get channel info. err: %v", err)
	// 	return
	// }
	// if cn.TMEnd < dbhandler.DefaultTimeStamp || cn.TMDelete < dbhandler.DefaultTimeStamp {
	// 	retryCount++
	// } else {
	// 	retryCount = 0
	// }

	// // if the retry count is bigger than defaultHealthMaxRetryCount,
	// // hangup the call
	// if retryCount > defaultHealthMaxRetryCount {
	// 	log.WithField("call", qc).Infof("Exceeded max call health check retry count. Hanging up the call. call_id: %s", qc.ID)
	// 	_, _ = h.HangingUp(ctx, qc.ID, call.HangupReasonNormal)
	// 	return
	// }

	// // send health check.
	// if errHealth := h.reqHandler.CallV1CallHealth(ctx, id, defaultHealthDelay, retryCount); errHealth != nil {
	// 	log.Errorf("Could not send the call health check request. err: %v", errHealth)
	// 	return
	// }
}
