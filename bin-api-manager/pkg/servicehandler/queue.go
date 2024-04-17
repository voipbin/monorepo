package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

// queueGet validates the queue's ownership and returns the queue info.
func (h *serviceHandler) queueGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*qmqueue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "queueGet",
		"customer_id": a.CustomerID,
		"queue_id":    id,
	})

	// send request
	res, err := h.reqHandler.QueueV1QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the queue info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", res).Debug("Received result.")

	return res, nil
}

// QueueGet sends a request to queue-manager
// to getting the queue.
func (h *serviceHandler) QueueGet(ctx context.Context, a *amagent.Agent, queueID uuid.UUID) (*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"queue_id":    queueID,
	})

	tmp, err := h.queueGet(ctx, a, queueID)
	if err != nil {
		log.Errorf("Could not validate the queue info. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueueGets sends a request to queue-manager
// to getting a list of queues.
// it returns queue info if it succeed.
func (h *serviceHandler) QueueGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	tmps, err := h.reqHandler.QueueV1QueueGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get queues from the queue-manager. err: %v", err)
		return nil, err
	}

	res := []*qmqueue.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// QueueCreate sends a request to queue-manager
// to creating an queue.
// it returns created queue info if it succeed.
func (h *serviceHandler) QueueCreate(
	ctx context.Context,
	a *amagent.Agent,
	name string,
	detail string,
	routingMethod qmqueue.RoutingMethod,
	tagIDs []uuid.UUID,
	waitActions []fmaction.Action,
	timeoutWait int,
	timeoutService int,
) (*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueueCreate(
		ctx,
		a.CustomerID,
		name,
		detail,
		qmqueue.RoutingMethod(routingMethod),
		tagIDs,
		waitActions,
		timeoutWait,
		timeoutService,
	)
	if err != nil {
		log.Errorf("Could not create the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debug("Create a new queue.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueueDelete sends a request to queue-manager
// to deleting the queue.
// it returns error if it failed.
func (h *serviceHandler) QueueDelete(ctx context.Context, a *amagent.Agent, queueID uuid.UUID) (*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	q, err := h.queueGet(ctx, a, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, q.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueueDelete(ctx, queueID)
	if err != nil {
		log.Errorf("Could not delete the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debugf("Deleted queue. queue_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueueUpdate sends a request to queue-manager
// to updating the queue.
// it returns error if it failed.
func (h *serviceHandler) QueueUpdate(
	ctx context.Context,
	a *amagent.Agent,
	queueID uuid.UUID,
	name string,
	detail string,
	routingMethod qmqueue.RoutingMethod,
	tagIDs []uuid.UUID,
	waitActions []fmaction.Action,
	timeoutWait int,
	serviceTimeout int,
) (*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "QueueUpdate",
		"customer_id":     a.CustomerID,
		"username":        a.Username,
		"name":            name,
		"detail":          detail,
		"routing_method":  routingMethod,
		"wait_actions":    waitActions,
		"wait_timeout":    timeoutWait,
		"service_timeout": serviceTimeout,
	})

	q, err := h.queueGet(ctx, a, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, q.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueueUpdate(ctx, queueID, name, detail, routingMethod, tagIDs, waitActions, timeoutWait, serviceTimeout)
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debugf("Updated queue. queue_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueueUpdateTagIDs sends a request to queue-manager
// to updating the queue's tag_ids.
// it returns error if it failed.
func (h *serviceHandler) QueueUpdateTagIDs(ctx context.Context, a *amagent.Agent, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueUpdateTagIDs",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	q, err := h.queueGet(ctx, a, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, q.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueueUpdateTagIDs(ctx, queueID, tagIDs)
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debugf("Updated queue. queue_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueueUpdateRoutingMethod sends a request to queue-manager
// to updating the queue's routing_method.
// it returns error if it failed.
func (h *serviceHandler) QueueUpdateRoutingMethod(ctx context.Context, a *amagent.Agent, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueUpdateRoutingMethod",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	q, err := h.queueGet(ctx, a, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, q.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueueUpdateRoutingMethod(ctx, queueID, routingMethod)
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debugf("Updated queue. queue_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueueUpdateActions sends a request to queue-manager
// to updating the queue's action settings.
// it returns error if it failed.
func (h *serviceHandler) QueueUpdateActions(ctx context.Context, a *amagent.Agent, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueUpdateActions",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	q, err := h.queueGet(ctx, a, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, q.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueueUpdateActions(ctx, queueID, waitActions, timeoutWait, timeoutService)
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debugf("Updated queue. queue_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
