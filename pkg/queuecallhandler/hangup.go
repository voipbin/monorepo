package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/notifyhandler"
)

// HangupByReferenceID handles reference's hangup.
func (h *queuecallHandler) Hangup(ctx context.Context, referenceID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Hangup",
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

	if errDel := h.queuecallReferenceHandler.Delete(ctx, qr.ID); errDel != nil {
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
	if qc.Status != queuecall.StatusWait {
		// nothing to do here
		return
	}

	if err := h.db.QueuecallDelete(ctx, qc.ID, queuecall.StatusAbandoned); err != nil {
		log.Errorf("Could not delete the queuecall. err: %v", err)
		return
	}

	// get updated queuecall and notify.
	tmp, err := h.db.QueuecallGet(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return
	}
	h.notifyhandler.NotifyEvent(ctx, notifyhandler.EventTypeQueuecallAbandoned, tmp.WebhookURI, tmp)

	// calculate the duration and increase the abandoned count
	duration := getDuration(ctx, tmp.TMCreate, tmp.TMDelete)
	if err := h.db.QueueIncreaseTotalAbandonedCount(ctx, tmp.QueueID, tmp.ID, duration); err != nil {
		log.Errorf("Could not increase the total abandoned count. err: %v", err)
		return
	}
}
