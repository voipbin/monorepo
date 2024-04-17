package queuecallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
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
	if errForward := h.reqHandler.FlowV1ActiveflowUpdateForwardActionID(ctx, qc.ReferenceActiveflowID, qc.ExitActionID, true); errForward != nil {
		log.Errorf("Could not forward the call. err: %v", errForward)
		return nil, errForward
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

	// send the forward request
	if errForward := h.reqHandler.FlowV1ActiveflowUpdateForwardActionID(ctx, qc.ReferenceActiveflowID, qc.ExitActionID, true); errForward != nil {
		// could not forward the call. but keep continuing
		log.Errorf("Could not forward the ca=o0mll. err: %v", errForward)
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
