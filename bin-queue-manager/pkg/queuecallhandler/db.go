package queuecallhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
)

// Gets returns queuecalls of the given customer_id
func (h *queuecallHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
	})

	res, err := h.db.QueuecallGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get queuecalls info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns queuecall info.
func (h *queuecallHandler) Get(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	res, err := h.db.QueuecallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get queuecall info.")
	}

	return res, nil
}

// GetByReferenceID returns queuecall info of the given referenceID.
func (h *queuecallHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error) {
	res, err := h.db.QueuecallGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get queuecall info of the given reference id")
	}

	return res, nil
}

// QueueCreate creates a new queue.
func (h *queuecallHandler) Create(
	ctx context.Context,
	q *queue.Queue,
	id uuid.UUID,
	referenceType queuecall.ReferenceType,
	referenceID uuid.UUID,
	referenceActiveflowID uuid.UUID,
	forwardActionID uuid.UUID,
	conferenceID uuid.UUID,
	source commonaddress.Address,
) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                    "Create",
		"queue_id":                q.ID,
		"reference_type":          referenceType,
		"reference_id":            referenceID,
		"reference_activeflow_id": referenceActiveflowID,
	})
	log.Debug("Creating a new queuecall.")

	// generate queue id
	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
	}
	log = log.WithField("id", id)

	qc := &queuecall.Queuecall{
		ID:         id,
		CustomerID: q.CustomerID,
		QueueID:    q.ID,

		ReferenceType:         referenceType,
		ReferenceID:           referenceID,
		ReferenceActiveflowID: referenceActiveflowID,

		ForwardActionID: forwardActionID,
		ConfbridgeID:    conferenceID,

		Source:        source,
		RoutingMethod: q.RoutingMethod,
		TagIDs:        q.TagIDs,

		Status:         queuecall.StatusInitiating,
		ServiceAgentID: uuid.Nil,

		TimeoutWait:    q.WaitTimeout,
		TimeoutService: q.ServiceTimeout,

		DurationWaiting: 0,
		DurationService: 0,
	}

	// create
	if err := h.db.QueuecallCreate(ctx, qc); err != nil {
		log.Errorf("Could not create a new queuecall. err: %v", err)
		return nil, err
	}

	// get created queuecall and notify
	res, err := h.db.QueuecallGet(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get created queuecall. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallCreated, res)

	if errSet := h.setVariables(ctx, q, res); errSet != nil {
		log.Errorf("Could not set variables. err: %v", errSet)
	}

	// send the queuecall timeout-wait
	if res.TimeoutWait > 0 {
		if errTiemout := h.reqHandler.QueueV1QueuecallTimeoutWait(ctx, res.ID, res.TimeoutWait); errTiemout != nil {
			log.Errorf("Could not send the timeout-wait request. err: %v", errTiemout)
		}
	}

	// start health check
	if errHealth := h.reqHandler.QueueV1QueuecallHealthCheck(ctx, res.ID, defaultHealthCheckDelay, 0); errHealth != nil {
		// could not start the health check, but just write the error message only.
		log.Errorf("Could not start health check. err: %v", errHealth)
	}

	return res, nil
}

// Delete deletes queuecall.
func (h *queuecallHandler) Delete(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Delete",
		"queuecall_id": id,
	})

	if errDelete := h.db.QueuecallDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the queuecall. err: %v", errDelete)
		return nil, errors.Wrap(errDelete, "Could not get queuecall info.")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted queuecall. err: %v", err)
		return nil, errors.Wrap(err, "Could not get deleted queuecall.")
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallDeleted, res)

	return res, nil
}

// UpdateStatusConnecting updates the queuecall's status to the connecting.
func (h *queuecallHandler) UpdateStatusConnecting(ctx context.Context, id uuid.UUID, agentID uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateStatusConnecting",
		"queuecall_id": id,
		"agent_id":     agentID,
	})
	log.Debug("Creating a new queuecall.")

	if err := h.db.QueuecallSetStatusConnecting(ctx, id, agentID); err != nil {
		log.Errorf("Could not update the status to connecting. agent id. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueuecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallConnecting, res)

	return res, nil
}

// UpdateStatusService updates the queuecall's status to the service.
func (h *queuecallHandler) UpdateStatusService(ctx context.Context, qc *queuecall.Queuecall) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateStatusService",
		"queuecall_id": qc.ID,
	})
	log.Debug("Updating queuecall status to service.")

	curTime := h.utilHandler.TimeGetCurTime()
	duration := getDuration(ctx, qc.TMCreate, curTime)

	if errService := h.db.QueuecallSetStatusService(ctx, qc.ID, int(duration.Milliseconds()), curTime); errService != nil {
		log.Errorf("Could not update queuecall's status to service. err: %v", errService)
		return nil, errors.Wrap(errService, "Could not update queuecall's status to service.")
	}

	res, err := h.Get(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallServiced, res)

	_, err = h.queueHandler.AddServiceQueuecallID(ctx, res.QueueID, res.ID)
	if err != nil {
		log.Errorf("Could not add the service queuecall. queuecall_id: %s", res.ID)
	}

	// send the queuecall timeout-service if it exists
	if qc.TimeoutService > 0 {
		if errTimeout := h.reqHandler.QueueV1QueuecallTimeoutService(ctx, res.ID, res.TimeoutService); errTimeout != nil {
			log.Errorf("Could not send the timeout-service request. err: %v", errTimeout)
		}
	}

	return res, nil
}

// UpdateStatusAbandoned updates the queuecall's status to the abandoned.
func (h *queuecallHandler) UpdateStatusAbandoned(ctx context.Context, qc *queuecall.Queuecall) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateStatusAbandoned",
		"queuecall_id": qc.ID,
	})
	log.Debug("Updating queuecall status to waiting.")

	curTime := h.utilHandler.TimeGetCurTime()
	duration := getDuration(ctx, qc.TMCreate, curTime)

	if errService := h.db.QueuecallSetStatusAbandoned(ctx, qc.ID, int(duration.Milliseconds()), curTime); errService != nil {
		log.Errorf("Could not update queuecall's status to service. err: %v", errService)
		return nil, errors.Wrap(errService, "Could not update queuecall's status to service.")
	}

	res, err := h.Get(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallAbandoned, res)

	// remove the queuecall from the queue.
	q, err := h.queueHandler.RemoveQueuecallID(ctx, qc.QueueID, qc.ID)
	if err != nil {
		log.Errorf("Could not remove the queuecall from the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", q).Debugf("Removed queuecall from the queue. queue_id: %s, queuecall_id: %s", q.ID, qc.ID)

	// delete confbridge
	log.Debugf("Deleting confbridge. confbridge_id: %s", res.ConfbridgeID)
	cb, errDelete := h.reqHandler.CallV1ConfbridgeDelete(ctx, res.ConfbridgeID)
	if errDelete != nil {
		log.Errorf("Could not delete the confbridge. err: %v", errDelete)
	}
	log.WithField("confbridge", cb).Debugf("Deleted confbridge.")

	// delete variables
	if errVariables := h.deleteVariables(ctx, res); errVariables != nil {
		log.Errorf("Could not delete variables. err: %v", errVariables)
	}

	return res, nil
}

// UpdateStatusDone updates the queuecall's status to the done.
func (h *queuecallHandler) UpdateStatusDone(ctx context.Context, qc *queuecall.Queuecall) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateStatusDone",
		"queuecall_id": qc.ID,
	})
	log.Debug("Updating queuecall status to waiting.")

	curTime := h.utilHandler.TimeGetCurTime()
	duration := getDuration(ctx, qc.TMCreate, curTime)

	if errService := h.db.QueuecallSetStatusDone(ctx, qc.ID, int(duration.Milliseconds()), curTime); errService != nil {
		log.Errorf("Could not update queuecall's status to service. err: %v", errService)
		return nil, errors.Wrap(errService, "Could not update queuecall's status to service.")
	}

	res, err := h.Get(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallDone, res)

	// remove the queuecall from the queue.
	q, err := h.queueHandler.RemoveQueuecallID(ctx, qc.QueueID, qc.ID)
	if err != nil {
		log.Errorf("Could not remove the queuecall from the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", q).Debugf("Removed queuecall from the queue. queue_id: %s, queuecall_id: %s", q.ID, qc.ID)

	// delete confbridge
	log.Debugf("Deleting confbridge. confbridge_id: %s", res.ConfbridgeID)
	cb, err := h.reqHandler.CallV1ConfbridgeDelete(ctx, res.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not delete the confbridge. err: %v", err)
	}
	log.WithField("confbridge", cb).Debugf("Deleted confbridge.")

	// delete variables
	if errVariables := h.deleteVariables(ctx, res); errVariables != nil {
		log.Errorf("Could not delete variables. err: %v", errVariables)
	}

	return res, nil
}

// UpdateStatusWaiting updates the queuecall's status to the waiting.
func (h *queuecallHandler) UpdateStatusWaiting(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateStatusWaiting",
		"queuecall_id": id,
	})
	log.Debug("Updating queuecall status to waiting.")

	if err := h.db.QueuecallSetStatusWaiting(ctx, id); err != nil {
		log.Errorf("Could not update the status to connecting. agent id. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueuecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallWaiting, res)

	// send queue execute update request
	go func() {
		// add the queuecall to the queue.
		_, err = h.queueHandler.AddWaitQueueCallID(ctx, res.QueueID, res.ID)
		if err != nil {
			log.Errorf("Could not add the queuecall to the queue. err: %v", err)
		}
	}()

	return res, nil
}
