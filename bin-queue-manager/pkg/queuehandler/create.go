package queuehandler

import (
	"context"
	"fmt"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"

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
	waitFlowID uuid.UUID,
	waitTimeout int,
	serviceTimeout int,
) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Create",
		"customer_id":     customerID,
		"name":            name,
		"detail":          detail,
		"routing_method":  routingMethod,
		"tag_ids":         tagIDs,
		"wait_flow_id":    waitFlowID,
		"wait_timeout":    waitTimeout,
		"service_timeout": serviceTimeout,
	})
	log.Debug("Creating a new queue.")

	// check resource limit
	valid, err := h.reqHandler.BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeQueue)
	if err != nil {
		log.Errorf("Could not validate resource limit. err: %v", err)
		return nil, fmt.Errorf("could not validate resource limit: %w", err)
	}
	if !valid {
		log.Infof("Resource limit exceeded for customer. customer_id: %s", customerID)
		return nil, fmt.Errorf("resource limit exceeded")
	}

	// generate queue id
	id := h.utilHandler.UUIDCreate()
	log = log.WithField("queue_id", id)

	if routingMethod != queue.RoutingMethodRandom {
		return nil, fmt.Errorf("wrong routing_method. routing_method: %s", routingMethod)
	}

	// create a new queue
	a := &queue.Queue{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		Name:   name,
		Detail: detail,

		RoutingMethod: routingMethod,
		TagIDs:        tagIDs,

		Execute: queue.ExecuteStop,

		WaitFlowID:          waitFlowID,
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
