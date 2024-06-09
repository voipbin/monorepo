package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	amresource "monorepo/bin-agent-manager/models/resource"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// agentResourceGet returns given agent resource info.
// It takes a context, an agent pointer, and an agent ID as parameters.
// It returns a pointer to an amresource.Resource and an error.
// If the agent is not found or an error occurs during the request, it returns nil and the corresponding error.
func (h *serviceHandler) agentResourceGet(ctx context.Context, a *amagent.Agent, resourceID uuid.UUID) (*amresource.Resource, error) {
	// Create a logrus logger with fields for function name, customer ID, agent ID, and username.
	log := logrus.WithFields(logrus.Fields{
		"func":        "agentResourceGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"resource_id": resourceID,
	})

	// Send a request to the reqHandler to get the agent info using the provided resource ID.
	res, err := h.reqHandler.AgentV1ResourceGet(ctx, resourceID)
	if err != nil {
		// If an error occurs during the request, log the error and return nil and the error.
		log.Errorf("Could not get the agent resource info. err: %v", err)
		return nil, err
	}

	// If the request is successful, log the received result and return the result and nil.
	log.WithField("resource", res).Debug("Received result.")
	return res, nil
}

// AgentResourceGet sends a request to agent-manager
// to getting an agent resource.
func (h *serviceHandler) AgentResourceGet(ctx context.Context, a *amagent.Agent, resourceID uuid.UUID) (*amresource.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentResourceGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"resource_id": resourceID,
	})

	tmp, err := h.agentResourceGet(ctx, a, resourceID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if a.ID != tmp.OwnerID && !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentResourceGets sends a request to agent-manager
// to getting a list of agent resources.
// it returns list of agent resources if it succeed.
func (h *serviceHandler) AgentResourceGets(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*amresource.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "AgentResourceGets",
		"agent":   a,
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmps, err := h.reqHandler.AgentV1ResourceGets(ctx, token, size, filters)
	if err != nil {
		log.Infof("Could not get agents info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*amresource.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// AgentResourceDelete sends a request to agent-manager
// to delete the agent resource.
func (h *serviceHandler) AgentResourceDelete(ctx context.Context, a *amagent.Agent, resourceID uuid.UUID) (*amresource.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentResourceDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"resource_id": resourceID,
	})

	r, err := h.agentResourceGet(ctx, a, resourceID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if a.ID != r.OwnerID && !h.hasPermission(ctx, a, r.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.AgentV1ResourceDelete(ctx, resourceID)
	if err != nil {
		log.Infof("Could not delete the agent resource info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
