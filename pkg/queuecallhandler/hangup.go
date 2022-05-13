package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// Hungup handles reference's hungup.
func (h *queuecallHandler) Hungup(ctx context.Context, referenceID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Hungup",
			"reference_id": referenceID,
		},
	)

	// get queuecallreference
	qr, err := h.queuecallReferenceHandler.Get(ctx, referenceID)
	if err != nil {
		log.Debug("Could not get queuecallreference. Consider this reference has not in the queue service.")
		return
	}
	log.WithField("queuecallreference", qr).Debug("Found queuecallreference.")

	_, errDel := h.queuecallReferenceHandler.Delete(ctx, qr.ID)
	if errDel != nil {
		log.Errorf("Could not delete the queuecall reference. But keep moving on. err: %v", errDel)
	}

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, qr.CurrentQueuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return
	}
	log = log.WithField("queuecall_id", qc.ID)

	// check queuecall's status
	if qc.Status != queuecall.StatusWaiting && qc.Status != queuecall.StatusKicking {
		// nothing to do here
		return
	}

	curTime := dbhandler.GetCurTime()
	duration := getDuration(ctx, qc.TMCreate, curTime)
	log.Debug("Calculated duration. duration: %ld", duration.Milliseconds())
	if err := h.db.QueuecallSetDurationWaiting(ctx, qc.ID, int(duration.Milliseconds())); err != nil {
		log.Errorf("Could not update queuecall's duration_waiting. err: %v", err)
		return
	}

	if err := h.db.QueuecallDelete(ctx, qc.ID, queuecall.StatusAbandoned, curTime); err != nil {
		log.Errorf("Could not delete the queuecall. err: %v", err)
		return
	}

	// get updated queuecall and notify.
	tmp, err := h.db.QueuecallGet(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return
	}
	h.notifyhandler.PublishWebhookEvent(ctx, tmp.CustomerID, queuecall.EventTypeQueuecallAbandoned, tmp)

	if err := h.db.QueueIncreaseTotalAbandonedCount(ctx, tmp.QueueID, tmp.ID); err != nil {
		log.Errorf("Could not increase the total abandoned count. err: %v", err)
		return
	}
}
