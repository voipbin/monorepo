package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

// flowGet validates the flow's ownership and returns the flow info.
func (h *serviceHandler) flowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "flowGet",
		"flow_id": flowID,
	})

	// send request
	res, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get the billing account info. err: %v", err)
		return nil, err
	}
	log.WithField("flow", res).Debugf("Received result. flow_id: %s", res.ID)

	if res.TMDelete < defaultTimestamp {
		log.Debugf("Deleted billing_account. billing_account_id: %s", res.ID)
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// FlowCreate is a service handler for flow creation.
func (h *serviceHandler) FlowCreate(ctx context.Context, a *amagent.Agent, name, detail string, actions []fmaction.Action, persist bool) (*fmflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowCreate",
		"customer_id": a.CustomerID,
		"name":        name,
		"persist":     persist,
	})
	log.WithField("actions", actions).Debug("Creating a new flow.")

	if persist {
		// we are making a persist flow here.
		// need to check if the customer has permission to create a persist flow.
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			log.Info("The user has no permission.")
			return nil, fmt.Errorf("user has no permission")
		}
	} else {
		// we are making a non-persist flow here.
		// no need to check the agent's permission. checking the customer id is good enough
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
			log.Info("The user has no permission.")
			return nil, fmt.Errorf("user has no permission")
		}
	}

	tmp, err := h.reqHandler.FlowV1FlowCreate(ctx, a.CustomerID, fmflow.TypeFlow, name, detail, actions, persist)
	if err != nil {
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowDelete deletes the flow of the given id.
func (h *serviceHandler) FlowDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*fmflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"flow_id":     id,
	})
	log.Debug("Deleting a flow.")

	// get flow
	f, err := h.flowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flow info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.FlowV1FlowDelete(ctx, id)
	if err != nil {
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowGet gets the flow of the given id.
// It returns flow if it succeed.
func (h *serviceHandler) FlowGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*fmflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"flow_id":     id,
	})
	log.Debug("Getting a flow.")

	// get flow
	tmp, err := h.flowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flow info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowGets gets the list of flow of the given customer id.
// It returns list of flows if it succeed.
func (h *serviceHandler) FlowGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*fmflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a flow.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
		"type":        string(fmflow.TypeFlow),
	}

	// get flows
	flows, err := h.reqHandler.FlowV1FlowGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get flows info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flows info. err: %v", err)
	}

	// create result
	res := []*fmflow.WebhookMessage{}
	for _, f := range flows {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// FlowUpdate updates the flow info.
// It returns updated flow if it succeed.
func (h *serviceHandler) FlowUpdate(ctx context.Context, a *amagent.Agent, f *fmflow.Flow) (*fmflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowUpdate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"flow":        f.ID,
	})
	log.Debug("Updating a flow.")

	// get flows
	tmpFlow, err := h.reqHandler.FlowV1FlowGet(ctx, f.ID)
	if err != nil {
		log.Errorf("Could not get flows info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flows info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmpFlow.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.FlowV1FlowUpdate(ctx, f)
	if err != nil {
		logrus.Errorf("Could not update the flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// FlowUpdateActions updates the flow's actions.
// It returns updated flow if it succeed.
func (h *serviceHandler) FlowUpdateActions(ctx context.Context, a *amagent.Agent, flowID uuid.UUID, actions []fmaction.Action) (*fmflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FlowUpdateActions",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"flow_id":     flowID,
	})
	log.Debug("Updating a flow actions.")

	// get flows
	f, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get flows info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flows info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.FlowV1FlowUpdateActions(ctx, flowID, actions)
	if err != nil {
		logrus.Errorf("Could not update the flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
