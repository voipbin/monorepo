package queuehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

// Gets returns queues
func (h *queueHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*queue.Queue, error) {
	log := logrus.WithField("func", "Gets")

	res, err := h.db.QueueGets(ctx, customerID, size, token)
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
func (h *queueHandler) Delete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"queue_id": id,
	})
	log.Debug("Deleting the queue info.")

	if err := h.db.QueueDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the queue. err: %v", err)
		return err
	}

	return nil
}

// UpdateBasicInfo updates the queue's basic info.
func (h *queueHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail, webhookURI, webhookMethod string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateBasicInfo",
		"queue_id":     id,
		"queue_name":   name,
		"queue_detail": detail,
	})
	log.Debug("Updating the queue's basic info.")

	err := h.db.QueueSetBasicInfo(ctx, id, name, detail, webhookURI, webhookMethod)
	if err != nil {
		log.Errorf("Could not update the basic info. err: %v", err)
		return err
	}

	return nil
}

// UpdateTagIDs updates the queue's tags.
func (h *queueHandler) UpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateTagIDs",
		"queue_id": id,
	})
	log.Debug("Updating the queue's tag ids.")

	if err := h.db.QueueSetTagIDs(ctx, id, tagIDs); err != nil {
		log.Errorf("Could not set the tags. err: %v", err)
		return err
	}

	return nil
}

// UpdateRoutingMethod updates the queue's routing method.
func (h *queueHandler) UpdateRoutingMethod(ctx context.Context, id uuid.UUID, routingMEthod queue.RoutingMethod) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateRoutingMethod",
		"queue_id": id,
	})
	log.Debug("Updating the queue's routing method.")

	if err := h.db.QueueSetRoutingMethod(ctx, id, routingMEthod); err != nil {
		log.Errorf("Could not set the addresses. err: %v", err)
		return err
	}

	return nil
}

// UpdateWaitActionsAndTimeouts updates the queue's wait/service info.
func (h *queueHandler) UpdateWaitActionsAndTimeouts(ctx context.Context, id uuid.UUID, waitActions []fmaction.Action, waitTimeout, serviceTimeout int) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateWaitActionsAndTimeouts",
		"queue_id": id,
	})
	log.Debug("Updating the queue's wait actions and timeouts.")

	if err := h.db.QueueSetWaitActionsAndTimeouts(ctx, id, waitActions, waitTimeout, serviceTimeout); err != nil {
		log.Errorf("Could not set the status. err: %v", err)
		return err
	}

	return nil
}
