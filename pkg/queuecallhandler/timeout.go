package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// TimeoutWait kicks the cqueuecall if the queuecall's status is wait.
func (h *queuecallHandler) TimeoutWait(ctx context.Context, queuecallID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "TimeoutWait",
			"queuecall_id": queuecallID,
		},
	)

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return
	}

	if qc.Status != queuecall.StatusWait {
		log.Debugf("The queuecall status is not wait. Ignore the request. status: %s", qc.Status)
		return
	}

	// Kick the queuecall.
	tmp, err := h.Kick(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		return
	}
	log.WithField("queuecall", tmp).Debugf("Detail queuecall info.")
}

// TimeoutService kicks the cqueuecall if the queuecall's status is service.
func (h *queuecallHandler) TimeoutService(ctx context.Context, queuecallID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "TimeoutService",
			"queuecall_id": queuecallID,
		},
	)

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return
	}

	if qc.Status != queuecall.StatusService {
		log.Debugf("The queuecall status is not wait. Ignore the request. status: %s", qc.Status)
		return
	}

	// Kick the queuecall.
	tmp, err := h.Kick(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		return
	}
	log.WithField("queuecall", tmp).Debugf("Detail queuecall info.")

}
