package agenthandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"monorepo/bin-agent-manager/internal/config"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	"monorepo/bin-agent-manager/pkg/metricshandler"
	bmaccount "monorepo/bin-billing-manager/models/account"
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/models/agent"
)

const (
	passwordResetTokenTTL = time.Hour // 1 hour
	passwordResetTokenLen = 32        // 32 bytes = 64 hex chars
)

// List returns agents
func (h *agentHandler) List(ctx context.Context, size uint64, token string, filters map[agent.Field]any) ([]*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.dbList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByCustomerIDAndAddress retrieves a list of agents based on the provided customer ID and address.
// It uses the provided context for cancellation and timeout.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// customerID (uuid.UUID): The ID of the customer.
// addr (commonaddress.Address): The address to filter agents by.
//
// Returns:
// ([]*agent.Agent, error): A slice of pointers to agent.Agent structs representing the retrieved agents,
// and an error if any occurred during the operation. If no agents are found, an empty slice is returned.
func (h *agentHandler) GetByCustomerIDAndAddress(ctx context.Context, customerID uuid.UUID, addr *commonaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetByCustomerIDAndAddress",
		"customer_id": customerID,
		"address":     addr,
	})

	res, err := h.db.AgentGetByCustomerIDAndAddress(ctx, customerID, addr)
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
func (h *agentHandler) Create(ctx context.Context, customerID uuid.UUID, username string, password string, name string, detail string, ringMethod agent.RingMethod, permission agent.Permission, tags []uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"username":    username,
		"permission":  permission,
	})
	log.Debug("Creating a new user.")

	// check resource limit
	rpcStart := time.Now()
	valid, err := h.reqHandler.BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeAgent)
	metricshandler.RPCCallDuration.WithLabelValues("billing-manager", "AccountIsValidResourceLimitByCustomerID").Observe(float64(time.Since(rpcStart).Milliseconds()))
	if err != nil {
		metricshandler.RPCCallTotal.WithLabelValues("billing-manager", "AccountIsValidResourceLimitByCustomerID", "failure").Inc()
		log.Errorf("Could not validate resource limit. err: %v", err)
		return nil, fmt.Errorf("could not validate resource limit: %w", err)
	}
	metricshandler.RPCCallTotal.WithLabelValues("billing-manager", "AccountIsValidResourceLimitByCustomerID", "success").Inc()
	if !valid {
		log.Infof("Resource limit exceeded for customer. customer_id: %s", customerID)
		return nil, fmt.Errorf("resource limit exceeded")
	}

	// validate username
	if !h.utilHandler.EmailIsValid(username) {
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

	onlyAdmin := h.isOnlyAdmin(ctx, id)
	if onlyAdmin {
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
func (h *agentHandler) isOnlyAdmin(ctx context.Context, id uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":     "isOnlyAdmin",
		"agent_id": id,
	})

	// get agnet
	a, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return true
	}

	if !a.HasPermission(agent.PermissionCustomerAdmin) && !a.HasPermission(agent.PermissionProjectSuperAdmin) {
		// the agent has no admin permission. no need to check the other agent
		log.Debugf("The agent has no admin permission.")
		return false
	}

	// get agents
	filters := map[agent.Field]any{
		agent.FieldCustomerID: a.CustomerID,
		agent.FieldDeleted:    false,
	}

	agents, err := h.dbList(ctx, 1000, "", filters)
	if err != nil {
		log.Warnf("Could not get agents info while verifying other admin agents. Treating the given agent as sole admin and denying operation as a fail-safe. agent_id: %s, err: %v", id.String(), err)
		return true
	}

	// check that there is another admin agent
	for _, a := range agents {
		if a.ID == id {
			continue
		}

		if a.HasPermission(agent.PermissionCustomerAdmin) || a.HasPermission(agent.PermissionProjectSuperAdmin) {
			// found admin permission agent.
			// return the true
			return false
		}
	}

	return true
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
		metricshandler.LoginTotal.WithLabelValues("failure").Inc()
		log.Errorf("Could not logged in. err: %v", err)
		return nil, errors.Wrap(err, "could not logged in")
	}
	metricshandler.LoginTotal.WithLabelValues("success").Inc()

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

	if permission != agent.PermissionCustomerAdmin {
		onlyAdmin := h.isOnlyAdmin(ctx, id)
		if onlyAdmin {
			return nil, fmt.Errorf("the agent is the only admin")
		}
	}

	res, err := h.UpdatePermissionRaw(ctx, id, permission)
	if err != nil {
		return nil, errors.Wrap(err, "could not update the agent permission")
	}

	return res, nil
}

// UpdatePermissionRaw updates the agent's permission without performing
// admin validation checks (such as guest-agent or only-admin checks).
// Callers are responsible for ensuring any required permission validation
// before invoking this method.
func (h *agentHandler) UpdatePermissionRaw(ctx context.Context, id uuid.UUID, permission agent.Permission) (*agent.Agent, error) {
	res, err := h.dbUpdatePermission(ctx, id, permission)
	if err != nil {
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
		"func":      "UpdateAddresses",
		"agent_id":  id,
		"addresses": addresses,
	})
	log.Debug("Updating the agent's addresses.")

	// get agent
	a, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the agent. err: %v", err)
		return nil, errors.Wrap(err, "could not get the agent")
	}
	log.WithField("agent", a).Debugf("Found agent info. agent_id: %s", a.ID)

	// validate the addresses
	for _, address := range addresses {
		// validate address
		switch address.Type {
		case commonaddress.TypeExtension:
			extensionID := uuid.FromStringOrNil(address.Target)
			if extensionID == uuid.Nil {
				return nil, errors.Errorf("invalid extension id")
			}

			rpcStart := time.Now()
			tmp, err := h.reqHandler.RegistrarV1ExtensionGet(ctx, extensionID)
			metricshandler.RPCCallDuration.WithLabelValues("registrar-manager", "ExtensionGet").Observe(float64(time.Since(rpcStart).Milliseconds()))
			if err != nil {
				metricshandler.RPCCallTotal.WithLabelValues("registrar-manager", "ExtensionGet", "failure").Inc()
				log.Errorf("Could not get extension info. err: %v", err)
				return nil, errors.Wrap(err, "could not get extension info")
			}
			metricshandler.RPCCallTotal.WithLabelValues("registrar-manager", "ExtensionGet", "success").Inc()

			if tmp.CustomerID != a.CustomerID {
				log.Errorf("Wrong customer info.")
				return nil, errors.Errorf("wrong customer info")
			}

		case commonaddress.TypeTel, commonaddress.TypeSIP:
			// validate tel/sip
			if len(address.Target) == 0 {
				return nil, errors.Errorf("invalid target")
			}

		default:
			return nil, errors.Errorf("unknown address type")
		}

		// check if the address is already assigned to the other agent
		ag, err := h.GetByCustomerIDAndAddress(ctx, a.CustomerID, &address)
		if err != nil && err != dbhandler.ErrNotFound {
			log.Errorf("Could not get agent info of the address. err: %v", err)
			return nil, errors.Wrap(err, "could not get agent info of the address")
		}

		if ag != nil && ag.ID != a.ID {
			log.Errorf("The address is already assigned to the other agent. agent_id: %s", ag.ID)
			return nil, errors.Errorf("the address is already assigned to the other agent")
		}
	}

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

// PasswordForgot generates a password reset token for the given username
// and sends the appropriate email (forgot-password or welcome).
func (h *agentHandler) PasswordForgot(ctx context.Context, username string, emailType PasswordResetEmailType) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "PasswordForgot",
		"username":  username,
		"emailType": emailType,
	})
	log.Debug("Processing password forgot request.")

	a, err := h.db.AgentGetByUsername(ctx, username)
	if err != nil {
		log.Infof("Could not find agent by username. err: %v", err)
		return fmt.Errorf("agent not found")
	}
	log.WithField("agent_id", a.ID).Debugf("Found agent for password reset. agent_id: %s", a.ID)

	tokenBytes := make([]byte, passwordResetTokenLen)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Errorf("Could not generate random token. err: %v", err)
		return errors.Wrap(err, "could not generate token")
	}
	token := hex.EncodeToString(tokenBytes)

	if err := h.cache.PasswordResetTokenSet(ctx, token, a.ID, passwordResetTokenTTL); err != nil {
		log.Errorf("Could not store password reset token. err: %v", err)
		return errors.Wrap(err, "could not store token")
	}

	cfg := config.Get()
	resetLink := cfg.PasswordResetBaseURL + "/auth/password-reset?token=" + token

	var subject, content string
	switch emailType {
	case PasswordResetEmailTypeWelcome:
		subject = "Welcome to VoIPBin - Set Your Password"
		content = fmt.Sprintf(
			"Welcome to VoIPBin! Your account has been created.\n\n"+
				"Click the link below to set your password. This link expires in 1 hour.\n\n"+
				"%s\n\n"+
				"If you have any questions, please contact our support team.",
			resetLink,
		)
	default:
		subject = "VoIPBin Password Reset"
		content = fmt.Sprintf(
			"You have requested a password reset for your VoIPBin account.\n\n"+
				"Click the link below to reset your password. This link expires in 1 hour.\n\n"+
				"%s\n\n"+
				"If you did not request this, you can safely ignore this email.",
			resetLink,
		)
	}

	destinations := []commonaddress.Address{
		{
			Type:   commonaddress.TypeEmail,
			Target: username,
		},
	}

	rpcStart := time.Now()
	_, err = h.reqHandler.EmailV1EmailSend(ctx, uuid.Nil, uuid.Nil, destinations, subject, content, nil)
	metricshandler.RPCCallDuration.WithLabelValues("email-manager", "EmailSend").Observe(float64(time.Since(rpcStart).Milliseconds()))
	if err != nil {
		metricshandler.RPCCallTotal.WithLabelValues("email-manager", "EmailSend", "failure").Inc()
		log.Errorf("Could not send password reset email. err: %v", err)
		return errors.Wrap(err, "could not send password reset email")
	}
	metricshandler.RPCCallTotal.WithLabelValues("email-manager", "EmailSend", "success").Inc()

	return nil
}

// PasswordReset validates the token and updates the agent's password.
func (h *agentHandler) PasswordReset(ctx context.Context, token string, password string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "PasswordReset",
	})
	log.Debug("Processing password reset request.")

	if len(password) < 8 {
		metricshandler.PasswordResetTotal.WithLabelValues("failure").Inc()
		return fmt.Errorf("password must be at least 8 characters")
	}

	agentID, err := h.cache.PasswordResetTokenGet(ctx, token)
	if err != nil {
		metricshandler.PasswordResetTotal.WithLabelValues("failure").Inc()
		log.Infof("Could not find password reset token. err: %v", err)
		return fmt.Errorf("invalid or expired token")
	}
	log.WithField("agent_id", agentID).Debugf("Found agent for token. agent_id: %s", agentID)

	if agentID == agent.GuestAgentID {
		metricshandler.PasswordResetTotal.WithLabelValues("failure").Inc()
		log.Infof("Attempted password reset for guest agent.")
		return fmt.Errorf("cannot reset password for guest agent")
	}

	passHash, err := h.utilHandler.HashGenerate(password, defaultPasswordHashCost)
	if err != nil {
		metricshandler.PasswordResetTotal.WithLabelValues("failure").Inc()
		log.Errorf("Could not generate password hash. err: %v", err)
		return errors.Wrap(err, "could not generate password hash")
	}

	if err := h.db.AgentSetPasswordHash(ctx, agentID, passHash); err != nil {
		metricshandler.PasswordResetTotal.WithLabelValues("failure").Inc()
		log.Errorf("Could not update password. err: %v", err)
		return errors.Wrap(err, "could not update password")
	}
	metricshandler.PasswordResetTotal.WithLabelValues("success").Inc()

	if err := h.cache.PasswordResetTokenDelete(ctx, token); err != nil {
		log.Errorf("Could not delete password reset token. err: %v", err)
	}

	res, err := h.db.AgentGet(ctx, agentID)
	if err != nil {
		log.Errorf("Could not get updated agent. err: %v", err)
		return nil
	}
	h.notifyHandler.PublishEvent(ctx, agent.EventTypeAgentUpdated, res)
	metricshandler.EventPublishTotal.WithLabelValues(string(agent.EventTypeAgentUpdated)).Inc()

	return nil
}
