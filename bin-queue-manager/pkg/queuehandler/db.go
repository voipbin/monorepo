package queuehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

// Gets returns queues
func (h *queueHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*queue.Queue, error) {
	log := logrus.WithField("func", "Gets")

	res, err := h.db.QueueGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get queues info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns queue info.
func (h *queueHandler) Get(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete updates the queue's basic info.
func (h *queueHandler) dbDelete(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"queue_id": id,
	})
	log.Debug("Deleting the queue info.")

	if err := h.db.QueueDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the queue. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted queue. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueDeleted, res)

	return res, nil
}

// UpdateBasicInfo updates the queue's basic info.
func (h *queueHandler) UpdateBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	routingMethod queue.RoutingMethod,
	tagIDs []uuid.UUID,
	waitActions []fmaction.Action,
	waitTimeout int,
	serviceTimeout int,
) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateBasicInfo",
		"queue_id":     id,
		"queue_name":   name,
		"queue_detail": detail,
	})
	log.Debug("Updating the queue's basic info.")

	err := h.db.QueueSetBasicInfo(
		ctx,
		id,
		name,
		detail,
		routingMethod,
		tagIDs,
		waitActions,
		waitTimeout,
		serviceTimeout,
	)
	if err != nil {
		log.Errorf("Could not update the basic info. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}

// UpdateTagIDs updates the queue's tags.
func (h *queueHandler) UpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateTagIDs",
		"queue_id": id,
	})
	log.Debug("Updating the queue's tag ids.")

	if err := h.db.QueueSetTagIDs(ctx, id, tagIDs); err != nil {
		log.Errorf("Could not set the tags. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}

// UpdateRoutingMethod updates the queue's routing method.
func (h *queueHandler) UpdateRoutingMethod(ctx context.Context, id uuid.UUID, routingMethod queue.RoutingMethod) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateRoutingMethod",
		"queue_id": id,
	})
	log.Debug("Updating the queue's routing method.")

	if err := h.db.QueueSetRoutingMethod(ctx, id, routingMethod); err != nil {
		log.Errorf("Could not set the addresses. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}

// UpdateWaitActionsAndTimeouts updates the queue's wait/service info.
func (h *queueHandler) UpdateWaitActionsAndTimeouts(ctx context.Context, id uuid.UUID, waitActions []fmaction.Action, waitTimeout, serviceTimeout int) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateWaitActionsAndTimeouts",
		"queue_id": id,
	})
	log.Debug("Updating the queue's wait actions and timeouts.")

	if err := h.db.QueueSetWaitActionsAndTimeouts(ctx, id, waitActions, waitTimeout, serviceTimeout); err != nil {
		log.Errorf("Could not set the status. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}

// UpdateExecute updates the queue's execute.
func (h *queueHandler) UpdateExecute(ctx context.Context, id uuid.UUID, execute queue.Execute) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateExecute",
		"queue_id": id,
		"execute":  execute,
	})
	log.Debug("Updating the queue's execute.")

	q, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
		return nil, err
	}

	if q.Execute == execute {
		// already same execute. nothing to do.
		return q, nil
	}

	if err := h.db.QueueSetExecute(ctx, id, execute); err != nil {
		log.Errorf("Could not set the execute. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	if execute == queue.ExecuteRun && q.Execute == queue.ExecuteStop {
		log.Debugf("The queue execute need to be run.")
		_ = h.reqHandler.QueueV1QueueExecuteRun(ctx, id, 100)
	}

	return res, nil
}

// AddWaitQueueCallID adds the queuecall to the wait queuecall ids.
func (h *queueHandler) AddWaitQueueCallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AddWaitQueueCallID",
		"queue_id":     id,
		"queuecall_id": queuecallID,
	})

	if errAdd := h.db.QueueAddWaitQueueCallID(ctx, id, queuecallID); errAdd != nil {
		log.Errorf("Could not add the queuecall id to the queue. err: %v", errAdd)
		return nil, errors.Wrap(errAdd, "Could not add the queuecall id to the queue.")
	}

	res, err := h.UpdateExecute(ctx, id, queue.ExecuteRun)
	if err != nil {
		log.Errorf("Could not update queue execute info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}

// RemoveServiceQueuecallID removes the service queuecall from the queue.
func (h *queueHandler) RemoveServiceQueuecallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "RemoveServiceQueuecallID",
		"queue_id":     id,
		"queuecall_id": queuecallID,
	})

	if errAdd := h.db.QueueRemoveServiceQueueCall(ctx, id, queuecallID); errAdd != nil {
		log.Errorf("Could not remove the service queuecall id to the queue. err: %v", errAdd)
		return nil, errors.Wrap(errAdd, "Could not remove the service queuecall id to the queue.")
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}

// RemoveWaitQueuecallID removes the wait queuecall from the queue.
func (h *queueHandler) RemoveWaitQueuecallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "RemoveWaitQueuecallID",
		"queue_id":     id,
		"queuecall_id": queuecallID,
	})

	if errAdd := h.db.QueueRemoveWaitQueueCall(ctx, id, queuecallID); errAdd != nil {
		log.Errorf("Could not remove the queuecall id to the queue. err: %v", errAdd)
		return nil, errors.Wrap(errAdd, "Could not add the queuecall id to the queue.")
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}

// AddServiceQueuecallID adds the given queuecall id to the service queue call of the queue.
func (h *queueHandler) AddServiceQueuecallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AddServiceQueuecallID",
		"queue_id":     id,
		"queuecall_id": queuecallID,
	})

	if errIncrease := h.db.QueueIncreaseTotalServicedCount(ctx, id, queuecallID); errIncrease != nil {
		log.Errorf("Could not increase the total serviced count. err: %v", errIncrease)
		return nil, errors.Wrap(errIncrease, "Could not add the queuecall info to the service queue.")
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}

// RemoveQueuecallID removes the queuecall from the queue's wait queuecall ids and service queuecall ids.
func (h *queueHandler) RemoveQueuecallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "RemoveQueuecallID",
		"queue_id":     id,
		"queuecall_id": queuecallID,
	})

	// remove from the wait queuecall ids
	if err := h.db.QueueRemoveWaitQueueCall(ctx, id, queuecallID); err != nil {
		log.Errorf("Could not remove the queuecall id from to the wait queuecall ids. err: %v", err)
		return nil, errors.Wrap(err, "Could not remove the queuecall id from the wait queuecall ids.")
	}

	// remove from the service queuecall ids
	if err := h.db.QueueRemoveServiceQueueCall(ctx, id, queuecallID); err != nil {
		log.Errorf("Could not remove the queuecall id from to the service queuecall ids. err: %v", err)
		return nil, errors.Wrap(err, "Could not remove the queuecall id from the service queuecall ids.")
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, queue.EventTypeQueueUpdated, res)

	return res, nil
}
