package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// Joined handle the situation when the queuecall joined to the the queue's confbridge.
func (h *queuecallHandler) Joined(ctx context.Context, referenceID, confbridgeID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Leaved",
			"reference_id":  referenceID,
			"confbridge_id": confbridgeID,
		},
	)

	// get queuecallreference
	qr, err := h.queuecallReferenceHandler.Get(ctx, referenceID)
	if err != nil {
		log.Debugf("Could not get queuecallreference. err: %v", err)
		return
	}
	log.WithField("queuecallreference", qr).Debug("Found queuecall reference.")

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, qr.CurrentQueuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return
	}
	log = log.WithField("queuecall_id", qc.ID)
	log.WithField("queuecall", qc).Debug("Found queuecall.")

	// compare confbridge info
	if qc.ConfbridgeID != confbridgeID {
		log.WithField("queuecall", qc).Infof("The confbridge info incorrect. Ignore the request. confbridge: %s", confbridgeID)
		return
	}

	// calculate the duration and set the duration_service
	curTime := dbhandler.GetCurTime()
	duration := getDuration(ctx, qc.TMCreate, curTime)
	log.Debug("Calculated duration. duration: %ld", duration.Milliseconds())
	if err := h.db.QueuecallSetDurationWaiting(ctx, qc.ID, int(duration.Milliseconds())); err != nil {
		log.Errorf("Could not update queuecall's duration_waiting. err: %v", err)
		return
	}

	// set queuecall's status to the service
	if errSet := h.db.QueuecallSetStatusService(ctx, qc.ID, curTime); errSet != nil {
		log.WithField("queuecall", qc).Debugf("Could not update the queuecall's status. err: %v", errSet)
		return
	}

	tmp, err := h.db.QueuecallGet(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return
	}
	h.notifyhandler.PublishWebhookEvent(ctx, tmp.CustomerID, queuecall.EventTypeQueuecallServiced, tmp)

	// get wait duration and increase the serviced count
	waitDuration := getDuration(ctx, qc.TMCreate, qc.TMService)
	log.Debugf("Serviced queuecall's detail. waiting_time: %d", waitDuration.Milliseconds())

	if err := h.db.QueueIncreaseTotalServicedCount(ctx, qc.QueueID, qc.ID); err != nil {
		log.Errorf("Could not increase the total serviced count. wait_duration: %d, err: %v", waitDuration.Milliseconds(), err)
	}

	// send the queuecall timeout-service
	if qc.TimeoutService > 0 {
		if err := h.reqHandler.QMV1QueuecallTimeoutService(ctx, qc.ID, qc.TimeoutService); err != nil {
			log.Errorf("Could not send the timeout-service request. err: %v", err)
		}
	}

}
