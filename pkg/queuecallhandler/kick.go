package queuecallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Kick kicks the queuecall from the queue
func (h *queuecallHandler) Kick(ctx context.Context, queuecallID uuid.UUID) error {
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
		return err
	}

	if qc.Status == queuecall.StatusDone || qc.Status == queuecall.StatusAbandoned {
		log.Errorf("The queuecall has over already. status: %s", qc.Status)
		return fmt.Errorf("invalid queuecall status. status: %s", qc.Status)
	}

	// send the forward request
	if err := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ExitActionID, true); err != nil {
		log.Errorf("Could not forward the call. err: %v", err)
		return err
	}

	return nil
}

// KickByReferenceID kicks the queuecall of the give reference id from the queue
func (h *queuecallHandler) KickByReferenceID(ctx context.Context, referenceID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Leave",
			"reference_id": referenceID,
		},
	)

	// get queuecallreference
	qcr, err := h.queuecallReferenceHandler.Get(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get queuecall reference. err: %v", err)
		return err
	}
	log = log.WithField("queuecall_id", qcr.CurrentQueuecallID)

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, qcr.CurrentQueuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return err
	}

	// check the queuecall's status
	if qc.Status == queuecall.StatusDone || qc.Status == queuecall.StatusAbandoned {
		log.Errorf("The queuecall's status is not valid. status: %s", qc.Status)
		return fmt.Errorf("invalid queuecall status. status: %s", qc.Status)
	}

	// send the forward request
	if err := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ExitActionID, true); err != nil {
		log.Errorf("Could not forward the call. err: %v", err)
		return err
	}

	return nil
}
