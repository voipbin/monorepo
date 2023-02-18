package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// EventCallCallHungup handles call-manager call_hungup
func (h *queuecallHandler) EventCallCallHungup(ctx context.Context, referenceID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "EventCallCallHungup",
			"reference_id": referenceID,
		},
	)

	qc, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		// no queuecall exist. nothing to do
		return
	}

	if qc.TMEnd < dbhandler.DefaultTimeStamp || qc.Status == queuecall.StatusService {
		// already done or other handler will deal with it.
		// nothing to do.
		return
	}

	_, err = h.UpdateStatusAbandoned(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status abandoned.")
	}
}

// EventCallConfbridgeJoined handles call-manager confbridge_join
func (h *queuecallHandler) EventCallConfbridgeJoined(ctx context.Context, referenceID uuid.UUID, confbridgeID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "EventCallConfbridgeJoined",
			"reference_id":  referenceID,
			"confbridge_id": confbridgeID,
		},
	)

	qc, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		// no queuecall exist. nothing to do
		return
	}

	if qc.TMEnd < dbhandler.DefaultTimeStamp || qc.ConfbridgeID != confbridgeID {
		// already done or other handler will deal with it.
		// nothing to do.
		return
	}

	// update queuecall info
	res, err := h.UpdateStatusService(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status to service. err: %v", err)
		return
	}
	log.WithField("queuecall", res).Debugf("Updated queuecall status service. queuecall_id: %s", res.ID)
}

// EventCallConfbridgeLeaved handles call-manager confbridge_leaved
func (h *queuecallHandler) EventCallConfbridgeLeaved(ctx context.Context, referenceID uuid.UUID, confbridgeID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "EventCallConfbridgeLeaved",
			"reference_id":  referenceID,
			"confbridge_id": confbridgeID,
		},
	)

	// get queuecall
	qc, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		return
	}

	// validate the queuecall info
	if qc.ConfbridgeID != confbridgeID || qc.TMEnd < dbhandler.DefaultTimeStamp {
		// queuecall is not valid.
		return
	}

	// update queuecall status to done
	res, err := h.UpdateStatusDone(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status to done. err: %v", err)
		return
	}
	log.WithField("queuecall", res).Debugf("Updated queuecall status done. queuecall_id: %s", res.ID)
}
