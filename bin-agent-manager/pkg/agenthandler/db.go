package agenthandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/metricshandler"
)

// dbList returns agents
func (h *agentHandler) dbList(ctx context.Context, size uint64, token string, filters map[agent.Field]any) ([]*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "dbList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.db.AgentList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbGet returns agent info.
func (h *agentHandler) dbGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "dbGet",
		"id":   id,
	})

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbCreate creates a new agent.
func (h *agentHandler) dbCreate(ctx context.Context, customerID uuid.UUID, username string, password string, name string, detail string, ringMethod agent.RingMethod, permission agent.Permission, tags []uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbCreate",
		"customer_id": customerID,
		"username":    username,
		"permission":  permission,
	})
	log.Debug("Creating a new user.")

	if ringMethod != agent.RingMethodRingAll {
		log.Errorf("Unsupported ring method. Currently, support ringall only. ringMethod: %s", ringMethod)
		return nil, fmt.Errorf("wrong ring_method")
	}

	// generate hash password
	hashPassword, err := h.utilHandler.HashGenerate(password, defaultPasswordHashCost)
	if err != nil {
		log.Errorf("Could not generate hash. err: %v", err)
		return nil, err
	}

	id := h.utilHandler.UUIDCreate()
	a := &agent.Agent{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, agent.EventTypeAgentCreated, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentCreated)).Inc()

	log.WithField("agent", res).Debug("Created a new agent.")

	return res, nil
}

// dbDelete deletes the agent
func (h *agentHandler) dbDelete(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "dbDelete",
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, agent.EventTypeAgentDeleted, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentDeleted)).Inc()

	return res, nil
}

// dbLogin validate the username and password.
func (h *agentHandler) dbLogin(ctx context.Context, username string, password string) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "dbLogin",
		"username": username,
	})
	log.Debug("Agent login.")

	res, err := h.db.AgentGetByUsername(ctx, username)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, fmt.Errorf("no agent info")
	}

	if !h.utilHandler.HashCheckPassword(password, res.PasswordHash) {
		return nil, fmt.Errorf("wrong password")
	}

	return res, nil
}

// dbUpdateInfo updates the agent's basic info.
func (h *agentHandler) dbUpdateInfo(ctx context.Context, id uuid.UUID, name string, detail string, ringMethod agent.RingMethod) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbUpdateInfo",
		"id":          id,
		"name":        name,
		"detail":      detail,
		"ring_method": ringMethod,
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
	h.notifyHandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentUpdated)).Inc()

	return res, nil
}

// dbUpdatePassword updates the agent's password.
func (h *agentHandler) dbUpdatePassword(ctx context.Context, id uuid.UUID, password string) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "dbUpdatePassword",
		"agent_id": id,
	})
	log.Debug("Updating the agent's password.")

	passHash, err := h.utilHandler.HashGenerate(password, defaultPasswordHashCost)
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
	h.notifyHandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentUpdated)).Inc()

	return res, nil
}

// dbUpdatePermission updates the agent's permission.
func (h *agentHandler) dbUpdatePermission(ctx context.Context, id uuid.UUID, permission agent.Permission) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "dbUpdatePermission",
		"id":         id,
		"permission": permission,
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, agent.EventTypeAgentUpdated, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentUpdated)).Inc()

	return res, nil
}

// dbUpdateTagIDs updates the agent's tags.
func (h *agentHandler) dbUpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "dbUpdateTagIDs",
		"id":      id,
		"tag_ids": tagIDs,
	})
	log.Debug("Updating the agent tag.")

	if err := h.db.AgentSetTagIDs(ctx, id, tagIDs); err != nil {
		log.Errorf("Could not set the tags. err: %v", err)
		return nil, err
	}

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated agent. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentUpdated)).Inc()

	return res, nil
}

// dbUpdateAddresses updates the agent's addresses.
func (h *agentHandler) dbUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "dbUpdateAddresses",
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
	h.notifyHandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentUpdated)).Inc()

	return res, nil
}

// dbUpdateStatus updates the agent's status.
func (h *agentHandler) dbUpdateStatus(ctx context.Context, id uuid.UUID, status agent.Status) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "dbUpdateStatus",
		"id":     id,
		"status": status,
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, agent.EventTypeAgentStatusUpdated, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentStatusUpdated)).Inc()

	return res, nil
}
