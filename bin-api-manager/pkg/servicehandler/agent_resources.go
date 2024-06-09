package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	amresource "monorepo/bin-agent-manager/models/resource"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// agentResourceGet validates the agent's ownership and returns the agent info.
// It takes a context, an agent pointer, and an agent ID as parameters.
// It returns a pointer to an amresource.Resource and an error.
// If the agent is not found or an error occurs during the request, it returns nil and the corresponding error.
func (h *serviceHandler) agentResourceGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amresource.Resource, error) {
	// Create a logrus logger with fields for function name, customer ID, agent ID, and username.
	log := logrus.WithFields(logrus.Fields{
		"func":        "agentResourceGet",
		"customer_id": a.CustomerID,
		"agent_id":    id,
		"username":    a.Username,
	})

	// Send a request to the reqHandler to get the agent info using the provided resource ID.
	res, err := h.reqHandler.AgentV1ResourceGet(ctx, id)
	if err != nil {
		// If an error occurs during the request, log the error and return nil and the error.
		log.Errorf("Could not get the agent info. err: %v", err)
		return nil, err
	}

	// If the request is successful, log the received result and return the result and nil.
	log.WithField("resource", res).Debug("Received result.")
	return res, nil
}
