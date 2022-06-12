package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
)

// agentGet validates the agent's ownership and returns the agent info.
func (h *serviceHandler) agentGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*amagent.Agent, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "agentGet",
			"customer_id": u.ID,
			"agent_id":    id,
		},
	)

	// send request
	res, err := h.reqHandler.AMV1AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the agent info. err: %v", err)
		return nil, err
	}
	log.WithField("agent", res).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	return res, nil
}

// AgentCreate sends a request to agent-manager
// to creating an agent.
// it returns created agent info if it succeed.
func (h *serviceHandler) AgentCreate(
	u *cscustomer.Customer,
	username string,
	password string,
	name string,
	detail string,
	ringMethod amagent.RingMethod,
	permission amagent.Permission,
	tagIDs []uuid.UUID,
	addresses []commonaddress.Address,
) (*amagent.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentCreate",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	// send request
	log.Debug("Creating a new agent.")
	tmp, err := h.reqHandler.AMV1AgentCreate(ctx, 30, u.ID, username, password, name, detail, ringMethod, permission, tagIDs, addresses)
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
func (h *serviceHandler) AgentGet(u *cscustomer.Customer, agentID uuid.UUID) (*amagent.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"agent_id":    agentID,
	})

	tmp, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentGet sends a request to agent-manager
// to getting a list of agents.
// it returns agent info if it succeed.
func (h *serviceHandler) AgentGets(u *cscustomer.Customer, size uint64, token string, tagIDs []uuid.UUID, status amagent.Status) ([]*amagent.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = getCurTime()
	}

	// get agents
	var tmps []amagent.Agent
	var err error
	if len(tagIDs) > 0 && status != "" {
		tmps, err = h.reqHandler.AMV1AgentGetsByTagIDsAndStatus(ctx, u.ID, tagIDs, amagent.Status(status))
	} else if len(tagIDs) > 0 {
		tmps, err = h.reqHandler.AMV1AgentGetsByTagIDs(ctx, u.ID, tagIDs)
	} else {
		tmps, err = h.reqHandler.AMV1AgentGets(ctx, u.ID, token, size)
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
func (h *serviceHandler) AgentDelete(u *cscustomer.Customer, agentID uuid.UUID) (*amagent.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"agent_id":    agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.AMV1AgentDelete(ctx, agentID)
	if err != nil {
		log.Infof("Could not delete the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentDelete sends a request to call-manager
// to delete the agent.
func (h *serviceHandler) AgentLogin(customerID uuid.UUID, username, password string) (string, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentLogin",
		"customer_id": customerID,
		"username":    username,
		"password":    len(password),
	})

	// send request
	ag, err := h.reqHandler.AMV1AgentLogin(ctx, 30000, customerID, username, password)
	if err != nil {
		log.Warningf("Could not agent login. err: %v", err)
		return "", err
	}

	tmp := ag.ConvertWebhookMessage()
	serialized, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return "", err
	}

	token, err := middleware.GenerateToken("agent", string(serialized[:]))
	if err != nil {
		logrus.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return token, nil
}

// AgentUpdate sends a request to agent-manager
// to update the agent info.
func (h *serviceHandler) AgentUpdate(u *cscustomer.Customer, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdate",
		"customer_id": u.ID,
		"username":    u.Username,
		"agent_id":    agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.AMV1AgentUpdate(ctx, agentID, name, detail, ringMethod)
	if err != nil {
		log.Infof("Could not delete the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentUpdate sends a request to agent-manager
// to update the agent's addresses info.
func (h *serviceHandler) AgentUpdateAddresses(u *cscustomer.Customer, agentID uuid.UUID, addresses []commonaddress.Address) (*amagent.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdateAddresses",
		"customer_id": u.ID,
		"username":    u.Username,
		"agent_id":    agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.AMV1AgentUpdateAddresses(ctx, agentID, addresses)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentUpdateTagIDs sends a request to agent-manager
// to update the agent's tag_ids info.
func (h *serviceHandler) AgentUpdateTagIDs(u *cscustomer.Customer, agentID uuid.UUID, tagIDs []uuid.UUID) (*amagent.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdateTagIDs",
		"customer_id": u.ID,
		"username":    u.Username,
		"agent_id":    agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.AMV1AgentUpdateTagIDs(ctx, agentID, tagIDs)
	if err != nil {
		log.Infof("Could not update the agent tag ids. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AgentUpdateStatus sends a request to agent-manager
// to update the agent status info.
func (h *serviceHandler) AgentUpdateStatus(u *cscustomer.Customer, agentID uuid.UUID, status amagent.Status) (*amagent.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentUpdateStatus",
		"customer_id": u.ID,
		"username":    u.Username,
		"agent_id":    agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.AMV1AgentUpdateStatus(ctx, agentID, status)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
