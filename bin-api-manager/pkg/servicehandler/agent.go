package servicehandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// agentGet validates the agent's ownership and returns the agent info.
func (h *serviceHandler) agentGet(ctx context.Context, id uuid.UUID) (*amagent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "agentGet",
		"agent_id": id,
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
	tmp, err := h.reqHandler.AgentV1AgentCreate(ctx, 30000, a.CustomerID, username, password, name, detail, ringMethod, permission, tagIDs, addresses)
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

	tmp, err := h.agentGet(ctx, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if a.ID != agentID && !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentGets sends a request to agent-manager
// to getting a list of agents.
// it returns list of agents if it succeed.
func (h *serviceHandler) AgentList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "AgentGets",
		"agent":   a,
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertAgentFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	tmps, err := h.agentList(ctx, size, token, typedFilters)
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

// agentGets sends a request to agent-manager
// to getting a list of agents.
// it returns list of agents if it succeed.
func (h *serviceHandler) agentList(ctx context.Context, size uint64, token string, filters map[amagent.Field]any) ([]amagent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "AgentGets",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	res, err := h.reqHandler.AgentV1AgentList(ctx, token, size, filters)
	if err != nil {
		log.Infof("Could not get agents info. err: %v", err)
		return nil, err
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

	af, err := h.agentGet(ctx, agentID)
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

	af, err := h.agentGet(ctx, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if a.ID != agentID && !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.agentUpdate(ctx, agentID, name, detail, ringMethod)
	if err != nil {
		log.Infof("Could not update the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) agentUpdate(ctx context.Context, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.Agent, error) {
	res, err := h.reqHandler.AgentV1AgentUpdate(ctx, agentID, name, detail, ringMethod)
	if err != nil {
		return nil, err
	}

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

	af, err := h.agentGet(ctx, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.agentUpdateAddresses(ctx, agentID, addresses)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) agentUpdateAddresses(ctx context.Context, agentID uuid.UUID, addresses []commonaddress.Address) (*amagent.Agent, error) {
	res, err := h.reqHandler.AgentV1AgentUpdateAddresses(ctx, agentID, addresses)
	if err != nil {
		return nil, err
	}

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

	af, err := h.agentGet(ctx, agentID)
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

	af, err := h.agentGet(ctx, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if a.ID != agentID && !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.agentUpdateStatus(ctx, agentID, status)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) agentUpdateStatus(ctx context.Context, agentID uuid.UUID, status amagent.Status) (*amagent.Agent, error) {
	res, err := h.reqHandler.AgentV1AgentUpdateStatus(ctx, agentID, status)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AgentUpdatePermission sends a request to agent-manager
// to update the agent permission info.
func (h *serviceHandler) AgentUpdatePermission(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, permission amagent.Permission) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdatePermission",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"agent_id":    agentID,
	})

	af, err := h.agentGet(ctx, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}
	log.Debugf("Updating agent permission. agent_id: %s, permission: %d", agentID, permission)

	if permission&amagent.PermissionProjectAll != 0 {
		// the agent trying to set the project level permission.
		// the only project level permission owned agent can set the project level permission.
		if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionNone) {
			log.Debugf("The agent has no project level permission.")
			return nil, fmt.Errorf("user has no permission")
		}
	} else {
		// customer level permission set.
		if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			log.Debugf("The agent has no customer level permission.")
			return nil, fmt.Errorf("user has no permission")
		}
	}

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdatePermission(ctx, agentID, permission)
	if err != nil {
		log.Infof("Could not update the agent permission. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentUpdatePassword sends a request to agent-manager
// to update the agent password.
func (h *serviceHandler) AgentUpdatePassword(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, password string) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdatePassword",
		"agent":    a,
		"password": len(password),
	})

	af, err := h.agentGet(ctx, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}
	log.Debugf("Updating agent password. agent_id: %s, password: %d", agentID, len(password))

	if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Debugf("The agent has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.agentUpdatePassword(ctx, agentID, password)
	if err != nil {
		log.Infof("Could not update the agent password. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) agentUpdatePassword(ctx context.Context, agentID uuid.UUID, password string) (*amagent.Agent, error) {
	res, err := h.reqHandler.AgentV1AgentUpdatePassword(ctx, 30000, agentID, password)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// convertAgentFilters converts map[string]string to map[amagent.Field]any
func (h *serviceHandler) convertAgentFilters(filters map[string]string) (map[amagent.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, amagent.Agent{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[amagent.Field]any, len(typed))
	for k, v := range typed {
		result[amagent.Field(k)] = v
	}

	return result, nil
}
