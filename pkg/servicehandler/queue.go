package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

// queueGet validates the queue's ownership and returns the agent info.
func (h *serviceHandler) queueGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*qmqueue.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "queueGet",
			"customer_id": u.ID,
			"agent_id":    id,
		},
	)

	// send request
	tmp, err := h.reqHandler.QMV1QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the queue info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueueGet sends a request to queue-manager
// to getting the queue.
func (h *serviceHandler) QueueGet(u *cscustomer.Customer, queueID uuid.UUID) (*qmqueue.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"agent_id":    queueID,
	})

	res, err := h.queueGet(ctx, u, queueID)
	if err != nil {
		log.Errorf("Could not validate the queue info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// QueueGets sends a request to queue-manager
// to getting a list of queues.
// it returns queue info if it succeed.
func (h *serviceHandler) QueueGets(u *cscustomer.Customer, size uint64, token string) ([]*qmqueue.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = getCurTime()
	}

	tmps, err := h.reqHandler.QMV1QueueGets(ctx, u.ID, token, size)
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
	u *cscustomer.Customer,
	name string,
	detail string,
	webhookURI string,
	webhookMethod string,
	routingMethod string,
	tagIDs []uuid.UUID,
	waitActions []fmaction.Action,
	timeoutWait int,
	timeoutService int,
) (*qmqueue.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueCreate",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	tmp, err := h.reqHandler.QMV1QueueCreate(
		ctx,
		u.ID,
		name,
		detail,
		webhookURI,
		webhookMethod,
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
func (h *serviceHandler) QueueDelete(u *cscustomer.Customer, queueID uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueDelete",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	_, err := h.queueGet(ctx, u, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return err
	}

	if err := h.reqHandler.QMV1QueueDelete(ctx, queueID); err != nil {
		log.Errorf("Could not delete the queue. err: %v", err)
		return err
	}

	return nil
}

// QueueUpdate sends a request to queue-manager
// to updating the queue.
// it returns error if it failed.
func (h *serviceHandler) QueueUpdate(u *cscustomer.Customer, queueID uuid.UUID, name, detail, webhookURI, webhookMethod string) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueUpdate",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	_, err := h.queueGet(ctx, u, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return err
	}

	if err := h.reqHandler.QMV1QueueUpdate(ctx, queueID, name, detail, webhookURI, webhookMethod); err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		return err
	}

	return nil
}

// QueueUpdateTagIDs sends a request to queue-manager
// to updating the queue's tag_ids.
// it returns error if it failed.
func (h *serviceHandler) QueueUpdateTagIDs(u *cscustomer.Customer, queueID uuid.UUID, tagIDs []uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueUpdateTagIDs",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	_, err := h.queueGet(ctx, u, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return err
	}

	if err := h.reqHandler.QMV1QueueUpdateTagIDs(ctx, queueID, tagIDs); err != nil {
		log.Errorf("Could not update the queue's tag_ids. err: %v", err)
		return err
	}

	return nil
}

// QueueUpdateRoutingMethod sends a request to queue-manager
// to updating the queue's routing_method.
// it returns error if it failed.
func (h *serviceHandler) QueueUpdateRoutingMethod(u *cscustomer.Customer, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueUpdateRoutingMethod",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	_, err := h.queueGet(ctx, u, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return err
	}

	if err := h.reqHandler.QMV1QueueUpdateRoutingMethod(ctx, queueID, routingMethod); err != nil {
		log.Errorf("Could not update the queue's routing_method. err: %v", err)
		return err
	}

	return nil
}

// QueueUpdateActions sends a request to queue-manager
// to updating the queue's action settings.
// it returns error if it failed.
func (h *serviceHandler) QueueUpdateActions(u *cscustomer.Customer, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueUpdateActions",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	_, err := h.queueGet(ctx, u, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return err
	}

	if err := h.reqHandler.QMV1QueueUpdateActions(ctx, queueID, waitActions, timeoutWait, timeoutService); err != nil {
		log.Errorf("Could not update the queue's actions. err: %v", err)
		return err
	}

	return nil
}
