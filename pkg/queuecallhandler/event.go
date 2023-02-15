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
		// no queuecall exist
		return
	}

	// validate queuecall
	if !h.validateForCallHungup(qc) {
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
		log.Errorf("Could not get queuecall. err: %v", err)
		return
	}
	log = log.WithField("queuecall_id", qc.ID)
	log.WithField("queuecall", qc).Debug("Found queuecall.")

	if qc.TMEnd < dbhandler.DefaultTimeStamp {
		log.Errorf("Unexepcted queuecall joined. queuecall_id: %s", qc.ID)
		// already done
		return
	}

	// compare confbridge info
	if qc.ConfbridgeID != confbridgeID {
		log.WithField("queuecall", qc).Infof("The conference info incorrect. Ignore the request. conference_id: %s", confbridgeID)
		return
	}

	// update queuecall info
	qc, err = h.UpdateStatusService(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status to service. err: %v", err)
		return
	}

	// send the queuecall timeout-service if it exists
	if qc.TimeoutService > 0 {
		if errTimeout := h.reqHandler.QueueV1QueuecallTimeoutService(ctx, qc.ID, qc.TimeoutService); errTimeout != nil {
			log.Errorf("Could not send the timeout-service request. err: %v", errTimeout)
		}
	}
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
		log.Errorf("Could not get queuecall. err: %v", err)
		return
	}
	log = log.WithField("queuecall_id", qc.ID)

	// validate the queuecall info
	if qc.ConfbridgeID != confbridgeID || qc.TMEnd < dbhandler.DefaultTimeStamp {
		// queuecall is not valid.
		return
	}
	log.WithField("queuecall", qc).Debugf("Found queuecall info. queuecall_id: %s", qc.ID)

	// we are expecting the queuecall's status was being serviced here.
	// because the only serviced status can be leave the queue.
	if qc.Status != queuecall.StatusService {
		log.Errorf("Invalid status. status: %s", qc.Status)
		return
	}

	// update queuecall status to done
	qc, err = h.UpdateStatusDone(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status to done. err: %v", err)
		return
	}

	// delete variables
	if errVariables := h.deleteVariables(ctx, qc); errVariables != nil {
		log.Errorf("Could not delete variables. err: %v", errVariables)
	}
}

func (h *queuecallHandler) validateForCallHungup(qc *queuecall.Queuecall) bool {
	if qc.TMEnd < dbhandler.DefaultTimeStamp {
		// queuecall already done
		return false
	}

	if qc.Status == queuecall.StatusService {
		// call-manager's confbridge leaved event handler will handle
		return false
	}

	return true
}
