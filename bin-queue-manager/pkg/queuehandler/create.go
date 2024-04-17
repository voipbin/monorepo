package queuehandler

import (
	"context"
	"fmt"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
)

// Create creates a new queue.
// waitTimeout: wait timeout(MS)
// serviceTimeout: service timeout(MS)
func (h *queueHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	routingMethod queue.RoutingMethod,
	tagIDs []uuid.UUID,
	waitActions []fmaction.Action,
	waitTimeout int,
	serviceTimeout int,
) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})
	log.Debug("Creating a new queue.")

	// generate queue id
	id := h.utilHandler.UUIDCreate()
	log = log.WithField("queue_id", id)

	if routingMethod != queue.RoutingMethodRandom {
		log.Errorf("Unsupported routing method. Currently, support random only. routingMethod: %s", routingMethod)
		return nil, fmt.Errorf("wrong routing_method")
	}

	// create a new queue
	a := &queue.Queue{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		RoutingMethod: routingMethod,
		TagIDs:        tagIDs,

		Execute: queue.ExecuteStop,

		WaitActions:         waitActions,
		WaitQueuecallIDs:    []uuid.UUID{},
		WaitTimeout:         waitTimeout,
		ServiceQueuecallIDs: []uuid.UUID{},
		ServiceTimeout:      serviceTimeout,

		TotalIncomingCount:  0,
		TotalServicedCount:  0,
		TotalAbandonedCount: 0,
	}

	if err := h.db.QueueCreate(ctx, a); err != nil {
		log.Errorf("Could not create a new queue. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created queue info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", res).Debug("Created a new queue.")

	return res, nil
}
