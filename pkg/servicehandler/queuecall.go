package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// queuecallGet validates the queuecall's ownership and returns the queuecall info.
func (h *serviceHandler) queuecallGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*qmqueuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "queuecallGet",
		"customer_id": a.CustomerID,
		"agent_id":    id,
	})

	// send request
	res, err := h.reqHandler.QueueV1QueuecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the queuecall info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", res).Debug("Received result.")

	return res, nil
}

// queuecallGetByReferenceID validates the queuecall's ownership and returns the queuecall info of the given reference id.
func (h *serviceHandler) queuecallGetByReferenceID(ctx context.Context, a *amagent.Agent, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "queuecallGetByReferenceID",
		"customer_id": a.CustomerID,
		"agent_id":    referenceID,
	})

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	res, err := h.reqHandler.QueueV1QueuecallGetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get the queuecall info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", res).Debug("Received result.")

	return res, nil
}

// QueuecallGet sends a request to queue-manager
// to getting the queuecall.
func (h *serviceHandler) QueuecallGet(ctx context.Context, a *amagent.Agent, queueID uuid.UUID) (*qmqueuecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueueGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    queueID,
	})

	tmp, err := h.queuecallGet(ctx, a, queueID)
	if err != nil {
		log.Errorf("Could not validate the queue info. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueuecallGets sends a request to queue-manager
// to getting a list of queuecalls.
// it returns queuecall info if it succeed.
func (h *serviceHandler) QueuecallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*qmqueuecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueuecallGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	tmps, err := h.reqHandler.QueueV1QueuecallGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get queues from the queue-manager. err: %v", err)
		return nil, err
	}

	res := []*qmqueuecall.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// QueuecallDelete sends a request to the queue-manager
// to delets the given queuecall.
func (h *serviceHandler) QueuecallDelete(ctx context.Context, a *amagent.Agent, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueuecallDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	qc, err := h.queuecallGet(ctx, a, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, qc.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueuecallDelete(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not delete the queuecall. err: %v", err)
		return nil, err
	}
	log.WithField("queuecall", tmp).Debugf("Deleted queuecall. queuecall_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueuecallKick sends a request to the queue-manager
// to kick out the given queuecall.
func (h *serviceHandler) QueuecallKick(ctx context.Context, a *amagent.Agent, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueuecallKick",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	qc, err := h.queuecallGet(ctx, a, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, qc.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueuecallKick(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not kick out the queuecall. err: %v", err)
		return nil, err
	}
	log.WithField("queuecall", tmp).Debugf("Kicked queuecall. queuecall_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// QueuecallKickByReferenceID sends a request to the queue-manager
// to kick out the given reference id.
func (h *serviceHandler) QueuecallKickByReferenceID(ctx context.Context, a *amagent.Agent, referenceID uuid.UUID) (*qmqueuecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "QueuecallKickByReferenceID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	qc, err := h.queuecallGetByReferenceID(ctx, a, referenceID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return nil, err
	}

	// permission check
	if !h.hasPermission(ctx, a, qc.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.QueueV1QueuecallKick(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not kick out the queuecall. err: %v", err)
		return nil, err
	}
	log.WithField("queuecall", tmp).Debugf("Kicked queuecall. queuecall_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
