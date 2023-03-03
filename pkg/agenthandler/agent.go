package agenthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// Gets returns agents
func (h *agentHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*agent.Agent, error) {
	log := logrus.WithField("func", "Gets")

	res, err := h.db.AgentGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByTagIDs returns agents
func (h *agentHandler) GetsByTagIDs(ctx context.Context, customerID uuid.UUID, tags []uuid.UUID) ([]*agent.Agent, error) {
	log := logrus.WithField("func", "GetsByTags")

	agents, err := h.db.AgentGets(ctx, customerID, maxAgentCount, getCurTime())
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return nil, err
	}

	res := []*agent.Agent{}
	for _, a := range agents {
		for _, b := range a.TagIDs {
			if contains(tags, b) {
				res = append(res, a)
				break
			}
		}
	}

	return res, nil
}

// GetsByTagIDsAndStatus returns agent with given condition.
func (h *agentHandler) GetsByTagIDsAndStatus(ctx context.Context, customerID uuid.UUID, tags []uuid.UUID, status agent.Status) ([]*agent.Agent, error) {
	log := logrus.WithField("func", "GetsByTagIDsAndStatus")

	agents, err := h.db.AgentGets(ctx, customerID, maxAgentCount, getCurTime())
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	res := []*agent.Agent{}
	for _, a := range agents {
		if a.Status != status {
			continue
		}

		for _, b := range a.TagIDs {
			if contains(tags, b) {
				res = append(res, a)
				break
			}
		}
	}

	return res, nil
}

// Get returns agent info.
func (h *agentHandler) Get(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new agent.
func (h *agentHandler) Create(ctx context.Context, customerID uuid.UUID, username, password, name, detail string, ringMethod agent.RingMethod, permission agent.Permission, tags []uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"username":    username,
		"permission":  permission,
	})
	log.Debug("Creating a new user.")

	// get agent
	tmpAgent, err := h.db.AgentGetByUsername(ctx, customerID, username)
	if err == nil {
		log.WithField("agent", tmpAgent).Errorf("The agent is already exist.")
		return nil, fmt.Errorf("already exist")
	}

	if ringMethod != agent.RingMethodRingAll {
		log.Errorf("Unsupported ring method. Currently, support ringall only. ringMethod: %s", ringMethod)
		return nil, fmt.Errorf("wrong ring_method")
	}

	// generate hash password
	hashPassword, err := generateHash(password)
	if err != nil {
		log.Errorf("Could not generate hash. err: %v", err)
		return nil, err
	}

	id := uuid.Must(uuid.NewV4())
	a := &agent.Agent{
		ID:           id,
		CustomerID:   customerID,
		Username:     username,
		PasswordHash: hashPassword,

		Name:   name,
		Detail: detail,

		RingMethod: ringMethod,
		Status:     agent.StatusOffline,
		Permission: permission,
		TagIDs:     tags,
		Addresses:  addresses,
	}
	log = log.WithField("agent_id", id)

	if err := h.db.AgentCreate(ctx, a); err != nil {
		log.Errorf("Could not create a new agent. err: %v", err)
		return nil, err
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created agent info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, agent.EventTypeAgentCreated, res)

	log.WithField("agent", res).Debug("Created a new agent.")

	return res, nil
}

// Delete updates the agent's basic info.
func (h *agentHandler) Delete(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"agent_id": id,
	})
	log.Debug("Deleting the agent info.")

	if err := h.db.AgentDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the agent. err: %v", err)
		return nil, err
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted agent info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, agent.EventTypeAgentDeleted, res)

	return res, nil
}

// Login validate the username and password.
func (h *agentHandler) Login(ctx context.Context, customerID uuid.UUID, username, password string) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Login",
		"customer_id":    customerID,
		"agent_username": username,
	})
	log.Debug("Agent login.")

	res, err := h.db.AgentGetByUsername(ctx, customerID, username)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, fmt.Errorf("no agent info")
	}

	if !checkHash(password, res.PasswordHash) {
		return nil, fmt.Errorf("wrong password")
	}

	return res, nil
}

// UpdateBasicInfo updates the agent's basic info.
func (h *agentHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateBasicInfo",
		"agent_id":     id,
		"agent_name":   name,
		"agent_detail": detail,
	})
	log.Debug("Updating the agent's basic info.")

	if errUpdate := h.db.AgentSetBasicInfo(ctx, id, name, detail, ringMethod); errUpdate != nil {
		log.Errorf("Could not update the basic info. err: %v", errUpdate)
		return nil, errUpdate
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated agent. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, agent.EventTypeAgentDeleted, res)

	return res, nil
}

// UpdatePassword updates the agent's password.
func (h *agentHandler) UpdatePassword(ctx context.Context, id uuid.UUID, password string) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdatePassword",
		"agent_id": id,
	})
	log.Debug("Updating the agent's password.")

	passHash, err := generateHash(password)
	if err != nil {
		log.Errorf("Could not generate the password hash. err: %v", err)
		return nil, err
	}

	if err := h.db.AgentSetPasswordHash(ctx, id, passHash); err != nil {
		log.Errorf("Could not update the password. err: %v", err)
		return nil, err
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated agent. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)

	return res, nil
}

// UpdatePermission updates the agent's permission.
func (h *agentHandler) UpdatePermission(ctx context.Context, id uuid.UUID, permission agent.Permission) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdatePermission",
		"agent_id": id,
	})
	log.Debug("Updating the agent's permission'.")

	if err := h.db.AgentSetPermission(ctx, id, permission); err != nil {
		log.Errorf("Could not set the permission. err: %v", err)
		return nil, err
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated agent. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)

	return res, nil
}

// UpdateTagIDs updates the agent's tags.
func (h *agentHandler) UpdateTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateTagIDs",
		"agent_id": id,
	})
	log.Debug("Updating the agent tag.")

	if err := h.db.AgentSetTagIDs(ctx, id, tags); err != nil {
		log.Errorf("Could not set the tags. err: %v", err)
		return nil, err
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated agent. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)

	return res, nil
}

// UpdateAddresses updates the agent's addresses.
func (h *agentHandler) UpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateAddresses",
		"agent_id": id,
	})
	log.Debug("Updating the agent's addresses.")

	if err := h.db.AgentSetAddresses(ctx, id, addresses); err != nil {
		log.Errorf("Could not set the addresses. err: %v", err)
		return nil, err
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated agent. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)

	return res, nil
}

// UpdateStatus updates the agent's status.
func (h *agentHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status agent.Status) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateStatus",
		"agent_id": id,
	})
	log.Debug("Updating the agent's status.")

	if err := h.db.AgentSetStatus(ctx, id, status); err != nil {
		log.Errorf("Could not set the status. err: %v", err)
		return nil, err
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated agent info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, agent.EventTypeAgentStatusUpdated, res)

	return res, nil
}
