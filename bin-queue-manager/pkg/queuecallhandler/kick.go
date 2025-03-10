package queuecallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

// Kick kicks the queuecall from the queue
func (h *queuecallHandler) Kick(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Kick",
		"queuecall_id": id,
	})

	// get queuecall
	qc, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return nil, err
	}

	if qc.Status == queuecall.StatusDone || qc.Status == queuecall.StatusAbandoned {
		log.Errorf("The queuecall has over already. status: %s", qc.Status)
		return nil, fmt.Errorf("invalid queuecall status. status: %s", qc.Status)
	}

	// send the forward request
	if errStop := h.reqHandler.FlowV1ActiveflowServiceStop(ctx, qc.ReferenceActiveflowID, qc.ID); errStop != nil {
		return nil, errors.Wrapf(errStop, "Could not stop the activeflow. activeflow_id: %s", qc.ReferenceActiveflowID)
	}

	if qc.Status == queuecall.StatusService {
		// nothing to do more.
		// the call-manager's confbridge_leaved message event subscriber will handle it.
		return qc, nil
	}

	// update status to abandoned
	res, err := h.UpdateStatusAbandoned(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status to abandoned. err: %v", err)
		return nil, err
	}

	return res, nil
}

// KickByReferenceID kicks the queuecall of the give reference id from the queue
func (h *queuecallHandler) KickByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "KickByReferenceID",
		"reference_id": referenceID,
	})
	log.Debugf("Kicking the call. reference_id: %s", referenceID)

	qc, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get queuecall info. err: %v", err)
		return nil, err
	}

	if qc.TMEnd < dbhandler.DefaultTimeStamp {
		// already ended
		return qc, nil
	}

	res, err := h.Kick(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		return nil, err
	}

	return res, nil
}

// kickForce kicks the given queuecall in force
func (h *queuecallHandler) kickForce(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "kickForce",
		"queuecall_id": id,
	})

	// get queuecall
	qc, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return nil, err
	}

	if qc.Status == queuecall.StatusDone || qc.Status == queuecall.StatusAbandoned {
		log.Errorf("The queuecall has already over. status: %s", qc.Status)
		return nil, fmt.Errorf("already done")
	}

	if errStop := h.reqHandler.FlowV1ActiveflowServiceStop(ctx, qc.ReferenceActiveflowID, qc.ID); errStop != nil {
		log.Errorf("Could not stop the activeflow. err: %v", errStop)
	}

	var res *queuecall.Queuecall
	if qc.Status == queuecall.StatusService {
		res, err = h.UpdateStatusDone(ctx, qc)
	} else {
		res, err = h.UpdateStatusAbandoned(ctx, qc)
	}
	if err != nil {
		log.Errorf("Could not update the queuecall status. err: %v", err)
		return nil, err
	}

	return res, nil
}
