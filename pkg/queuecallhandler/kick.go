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
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Kick",
			"queuecall_id": id,
		},
	)

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
	if err := h.reqHandler.FlowV1ActiveflowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ExitActionID, true); err != nil {
		log.Errorf("Could not forward the call. err: %v", err)
		return nil, err
	}

	if qc.Status == queuecall.StatusService {
		// nothing to do here.
		// the call-manager's confbridge_leaved message event subscribe will handle it.
		return qc, nil
	}

	// update status to abandoned
	res, err := h.UpdateStatusAbandoned(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status to abandoned. err: %v", err)
		return nil, err
	}

	if errVariables := h.deleteVariables(ctx, res); errVariables != nil {
		log.Errorf("Could not delete variables. err: %v", errVariables)
	}

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
