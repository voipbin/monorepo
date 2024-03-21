package agenthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// Gets returns agents
func (h *agentHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.dbGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns agent info.
func (h *agentHandler) Get(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.dbGet(ctx, id)
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

	// validate username
	if !isEmailValid(username) {
		log.Infof("Wrong username type. The username must be email format. username: %s", username)
		return nil, fmt.Errorf("wrong username format")
	}

	// check existence
	tmpAgent, err := h.db.AgentGetByUsername(ctx, username)
	if err == nil {
		log.WithField("agent", tmpAgent).Errorf("The agent is already exist.")
		return nil, fmt.Errorf("already exist")
	}

	res, err := h.dbCreate(ctx, customerID, username, password, name, detail, ringMethod, permission, tags, addresses)
	if err != nil {
		log.Errorf("Could not create an agent. err: %v", err)
		return nil, errors.Wrap(err, "could not create an agent")
	}

	return res, nil
}

// Delete deletes the agent.
func (h *agentHandler) Delete(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"agent_id": id,
	})

	// check the agent is deletable
	if id == agent.GuestAgentID {
		return nil, errors.Errorf("agent is guest agent")
	}

	onlyAdmin, err := h.isOnlyAdmin(ctx, id)
	if err != nil {
		log.Errorf("Could not check the agent is the only admin. erR: %v", err)
		return nil, errors.Wrapf(err, "could not check the agent is the only admin")
	} else if onlyAdmin {
		return nil, errors.Errorf("the agent is the only admin")
	}

	res, err := h.deleteForce(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the agent. err: %v", err)
		return nil, errors.Wrap(err, "could not delete the agent")
	}

	return res, nil
}

// deleteForce deletes the agent without any condition checks.
func (h *agentHandler) deleteForce(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "deleteForce",
		"agent_id": id,
	})

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the agent. err: %v", err)
		return nil, errors.Wrap(err, "could not delete the agent")
	}

	return res, nil
}

// isOnlyAdmin returns true if the given agent is the only admin
func (h *agentHandler) isOnlyAdmin(ctx context.Context, id uuid.UUID) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "isDeletable",
		"agent_id": id,
	})

	// get agnet
	a, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return false, err
	}

	if !a.HasPermission(agent.PermissionCustomerAdmin) && !a.HasPermission(agent.PermissionProjectSuperAdmin) {
		// the agent has no admin permission. no need to check the other agent
		return false, nil
	}

	// get agents
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}

	agents, err := h.dbGets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return false, errors.Wrapf(err, "could not get agents info")
	}

	// check that there is another admin agent
	for _, a := range agents {
		if a.ID == id {
			continue
		}

		if a.HasPermission(agent.PermissionCustomerAdmin) || a.HasPermission(agent.PermissionProjectSuperAdmin) {
			// found admin permission agent.
			// return the true
			return false, nil
		}
	}

	return true, nil
}

// Login validate the username and password.
func (h *agentHandler) Login(ctx context.Context, username string, password string) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Login",
		"username": username,
	})
	log.Debug("Agent login.")

	res, err := h.dbLogin(ctx, username, password)
	if err != nil {
		log.Errorf("Could not logged in. err: %v", err)
		return nil, errors.Wrap(err, "could not logged in")
	}

	return res, nil
}

// UpdateBasicInfo updates the agent's basic info.
func (h *agentHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string, ringMethod agent.RingMethod) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateBasicInfo",
		"agent_id":     id,
		"agent_name":   name,
		"agent_detail": detail,
	})
	log.Debug("Updating the agent's basic info.")

	res, err := h.dbUpdateInfo(ctx, id, name, detail, ringMethod)
	if err != nil {
		log.Errorf("Could not update the agent info. err: %v", err)
		return nil, errors.Wrap(err, "could not update the agent info")
	}

	return res, nil
}

// UpdatePassword updates the agent's password.
func (h *agentHandler) UpdatePassword(ctx context.Context, id uuid.UUID, password string) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdatePassword",
		"agent_id": id,
	})
	log.Debug("Updating the agent's password.")

	if id == agent.GuestAgentID {
		return nil, errors.Errorf("agent is guest agent")
	}

	res, err := h.dbUpdatePassword(ctx, id, password)
	if err != nil {
		log.Errorf("Could not update the agent's password. err: %v", err)
		return nil, errors.Wrap(err, "could not update the agent's password")
	}

	return res, nil
}

// UpdatePermission updates the agent's permission.
func (h *agentHandler) UpdatePermission(ctx context.Context, id uuid.UUID, permission agent.Permission) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdatePermission",
		"agent_id": id,
	})
	log.Debug("Updating the agent's permission'.")

	if id == agent.GuestAgentID {
		return nil, fmt.Errorf("agent id is guest agent")
	}

	onlyAdmin, err := h.isOnlyAdmin(ctx, id)
	if err != nil {
		log.Errorf("Could not check the agent is the only admin. err: %v", err)
		return nil, errors.Wrapf(err, "could not check the agent is the only admin")
	} else if onlyAdmin {
		log.Debugf("The agent is the only admin. only_admin: %v", onlyAdmin)
		return nil, errors.Wrapf(err, "the agent is the only admin")
	}

	res, err := h.dbUpdatePermission(ctx, id, permission)
	if err != nil {
		log.Errorf("Could not update the agent permission. err: %v", err)
		return nil, errors.Wrap(err, "could not update the agent permission")
	}

	return res, nil
}

// UpdateTagIDs updates the agent's tags.
func (h *agentHandler) UpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateTagIDs",
		"agent_id": id,
		"tag_ids":  tagIDs,
	})
	log.Debug("Updating the agent tag.")

	res, err := h.dbUpdateTagIDs(ctx, id, tagIDs)
	if err != nil {
		log.Errorf("Could not update the tag ids. err: %v", err)
		return nil, errors.Wrap(err, "could not update the tag ids")
	}

	return res, nil
}

// UpdateAddresses updates the agent's addresses.
func (h *agentHandler) UpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateAddresses",
		"agent_id": id,
	})
	log.Debug("Updating the agent's addresses.")

	res, err := h.dbUpdateAddresses(ctx, id, addresses)
	if err != nil {
		log.Errorf("Could not update the addresses. err: %v", err)
		return nil, errors.Wrap(err, "could not update the addresses")
	}

	return res, nil
}

// UpdateStatus updates the agent's status.
func (h *agentHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status agent.Status) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "UpdateStatus",
		"agent_id": id,
	})
	log.Debug("Updating the agent's status.")

	res, err := h.dbUpdateStatus(ctx, id, status)
	if err != nil {
		log.Errorf("Could not update the agent's status. err: %v", err)
		return nil, errors.Wrap(err, "could not update the agent's status")
	}

	return res, nil
}
