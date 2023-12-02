package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// agentGet validates the agent's ownership and returns the agent info.
func (h *serviceHandler) agentGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amagent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "agentGet",
		"customer_id": a.CustomerID,
		"agent_id":    id,
		"username":    a.Username,
	})

	// send request
	res, err := h.reqHandler.AgentV1AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the agent info. err: %v", err)
		return nil, err
	}
	log.WithField("agent", res).Debug("Received result.")

	return res, nil
}

// AgentCreate sends a request to agent-manager
// to creating an agent.
// it returns created agent info if it succeed.
func (h *serviceHandler) AgentCreate(
	ctx context.Context,
	a *amagent.Agent,
	username string,
	password string,
	name string,
	detail string,
	ringMethod amagent.RingMethod,
	permission amagent.Permission,
	tagIDs []uuid.UUID,
	addresses []commonaddress.Address,
) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	log.Debug("Creating a new agent.")
	tmp, err := h.reqHandler.AgentV1AgentCreate(ctx, 30, a.CustomerID, username, password, name, detail, ringMethod, permission, tagIDs, addresses)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}
	log.WithField("agent", tmp).Debug("Received result.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentGet sends a request to agent-manager
// to getting an agent.
func (h *serviceHandler) AgentGet(ctx context.Context, a *amagent.Agent, agentID uuid.UUID) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    agentID,
	})

	tmp, err := h.agentGet(ctx, a, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentGet sends a request to agent-manager
// to getting a list of agents.
// it returns agent info if it succeed.
func (h *serviceHandler) AgentGets(ctx context.Context, a *amagent.Agent, size uint64, token string, tagIDs []uuid.UUID, status amagent.Status) ([]*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get agents
	var tmps []amagent.Agent
	var err error
	if len(tagIDs) > 0 && status != "" {
		tmps, err = h.reqHandler.AgentV1AgentGetsByTagIDsAndStatus(ctx, a.CustomerID, tagIDs, amagent.Status(status))
	} else if len(tagIDs) > 0 {
		tmps, err = h.reqHandler.AgentV1AgentGetsByTagIDs(ctx, a.CustomerID, tagIDs)
	} else {
		tmps, err = h.reqHandler.AgentV1AgentGets(ctx, a.CustomerID, token, size)
	}
	if err != nil {
		log.Infof("Could not get agents info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*amagent.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// AgentDelete sends a request to call-manager
// to delete the agent.
func (h *serviceHandler) AgentDelete(ctx context.Context, a *amagent.Agent, agentID uuid.UUID) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    agentID,
	})

	af, err := h.agentGet(ctx, a, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.AgentV1AgentDelete(ctx, agentID)
	if err != nil {
		log.Infof("Could not delete the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentUpdate sends a request to agent-manager
// to update the agent info.
func (h *serviceHandler) AgentUpdate(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    agentID,
	})

	af, err := h.agentGet(ctx, a, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdate(ctx, agentID, name, detail, ringMethod)
	if err != nil {
		log.Infof("Could not delete the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentUpdate sends a request to agent-manager
// to update the agent's addresses info.
func (h *serviceHandler) AgentUpdateAddresses(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, addresses []commonaddress.Address) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdateAddresses",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    agentID,
	})

	af, err := h.agentGet(ctx, a, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdateAddresses(ctx, agentID, addresses)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentUpdateTagIDs sends a request to agent-manager
// to update the agent's tag_ids info.
func (h *serviceHandler) AgentUpdateTagIDs(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, tagIDs []uuid.UUID) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdateTagIDs",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    agentID,
	})

	af, err := h.agentGet(ctx, a, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdateTagIDs(ctx, agentID, tagIDs)
	if err != nil {
		log.Infof("Could not update the agent tag ids. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentUpdateStatus sends a request to agent-manager
// to update the agent status info.
func (h *serviceHandler) AgentUpdateStatus(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, status amagent.Status) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdateStatus",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    agentID,
	})

	af, err := h.agentGet(ctx, a, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdateStatus(ctx, agentID, status)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
