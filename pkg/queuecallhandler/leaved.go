package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/notifyhandler"
)

// Leaved handle the situation when the queuecall left from the queue's confbridge.
func (h *queuecallHandler) Leaved(ctx context.Context, referenceID, confbridgeID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Leaved",
			"reference_id":  referenceID,
			"confbridge_id": confbridgeID,
		},
	)

	// get queuecallreference
	qm, err := h.queuecallReferenceHandler.Get(ctx, referenceID)
	if err != nil {
		log.Debugf("Could not get queuecallreference. err: %v", err)
		return
	}
	log.WithField("queuecallreference", qm).Debug("Found queuecall reference.")

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, qm.CurrentQueuecallID)
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

	// we are expecting the queuecall's status is being serviced here.
	// because the only serviced status can be leave the queue.
	if qc.Status != queuecall.StatusService {
		log.Errorf("Invalid status. status: %s", qc.Status)
		return
	}

	if err := h.db.QueuecallDelete(ctx, qc.ID, queuecall.StatusDone); err != nil {
		log.Errorf("Could not delete the queuecall. err: %v", err)
		return
	}

	// get updated queuecall and notify.
	tmp, err := h.db.QueuecallGet(ctx, qm.CurrentQueuecallID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return
	}
	h.notifyhandler.NotifyEvent(ctx, notifyhandler.EventTypeQueuecallDone, tmp.WebhookURI, tmp)

	// calculate the duration and increase the serviced count
	duration := getDuration(ctx, tmp.TMService, tmp.TMDelete)
	if err := h.db.QueueRemoveServiceQueueCall(ctx, tmp.QueueID, tmp.ID, duration); err != nil {
		log.Errorf("Could not remove the queuecall from the service queuecall. service_duration: %d, err: %v", duration.Milliseconds(), err)
		return
	}

}
