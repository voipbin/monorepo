package queuecallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Kick kicks the queuecall from the queue
func (h *queuecallHandler) Kick(ctx context.Context, queuecallID uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Kick",
			"queuecall_id": queuecallID,
		},
	)

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return nil, err
	}

	if qc.Status == queuecall.StatusDone || qc.Status == queuecall.StatusAbandoned {
		log.Errorf("The queuecall has over already. status: %s", qc.Status)
		return nil, fmt.Errorf("invalid queuecall status. status: %s", qc.Status)
	}

	// send the forward request
	if err := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ExitActionID, true); err != nil {
		log.Errorf("Could not forward the call. err: %v", err)
		return nil, err
	}

	if errStatus := h.db.QueuecallSetStatusKicking(ctx, queuecallID); errStatus != nil {
		log.Errorf("Could not update the queuecall status. err: %v", errStatus)
		return nil, err
	}

	res, err := h.db.QueuecallGet(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not get updated queucall. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallCreated, res)

	return res, nil
}

// KickByReferenceID kicks the queuecall of the give reference id from the queue
func (h *queuecallHandler) KickByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "KickByReferenceID",
			"reference_id": referenceID,
		},
	)
	log.Debugf("Kicking the call. reference_id: %s", referenceID)

	// get queuecallreference
	qcr, err := h.queuecallReferenceHandler.Get(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get queuecall reference. err: %v", err)
		return nil, err
	}
	log = log.WithField("queuecall_id", qcr.CurrentQueuecallID)

	res, err := h.Kick(ctx, qcr.CurrentQueuecallID)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		return nil, err
	}

	return res, nil
}
