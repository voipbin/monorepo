package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// EventCallCallHangup handles call-manager call_hungup
func (h *queuecallHandler) EventCallCallHangup(ctx context.Context, referenceID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "EventCallCallHangup",
		"reference_id": referenceID,
	})

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
	log := logrus.WithFields(logrus.Fields{
		"func":          "EventCallConfbridgeJoined",
		"reference_id":  referenceID,
		"confbridge_id": confbridgeID,
	})

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
	log := logrus.WithFields(logrus.Fields{
		"func":          "EventCallConfbridgeLeaved",
		"reference_id":  referenceID,
		"confbridge_id": confbridgeID,
	})

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

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *queuecallHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all queues in customer. customer_id: %s", cu.ID)

	// get all queuecalls in customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	qs, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets queuecalls list. err: %v", err)
		return errors.Wrap(err, "could not get queuecalls list")
	}

	// kick all queuecalls
	for _, q := range qs {
		log.Debugf("Kicking out the queuecalls from the queue. queuecall_id: %s", q.ID)
		qc, err := h.kickForce(ctx, q.ID)
		if err != nil {
			log.Errorf("Could not kick out the queuecall from the queue. err: %v", err)
			continue
		}
		log.WithField("queuecall", qc).Debugf("Kicked out the queuecall. queuecall_id: %s", qc.ID)
	}

	// delete all queuecalls
	for _, q := range qs {
		log.Debugf("Deleting queuecall info. queuecall_id: %s", q.ID)
		tmp, err := h.Delete(ctx, q.ID)
		if err != nil {
			log.Errorf("Could not delete queuecall info. err: %v", err)
			continue
		}
		log.WithField("queuecall", tmp).Debugf("Deleted queuecall info. queuecall_id: %s", tmp.ID)
	}

	return nil
}
