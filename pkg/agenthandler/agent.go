package agenthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentcall"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
)

// AgentGets returns agents
func (h *agentHandler) AgentGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*agent.Agent, error) {
	log := logrus.WithField("func", "AgentGets")

	res, err := h.db.AgentGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// AgentGetsByTagIDs returns agents
func (h *agentHandler) AgentGetsByTagIDs(ctx context.Context, customerID uuid.UUID, tags []uuid.UUID) ([]*agent.Agent, error) {
	log := logrus.WithField("func", "AgentGetsByTags")

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

// AgentGetsByTagIDsAndStatus returns agent with given condition.
func (h *agentHandler) AgentGetsByTagIDsAndStatus(ctx context.Context, customerID uuid.UUID, tags []uuid.UUID, status agent.Status) ([]*agent.Agent, error) {
	log := logrus.WithField("func", "AgentGetsByTagIDsAndStatus")

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

// AgentGet returns agent info.
func (h *agentHandler) AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithField("func", "AgentGet")

	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// AgentCreate creates a new agent.
func (h *agentHandler) AgentCreate(ctx context.Context, customerID uuid.UUID, username, password, name, detail string, ringMethod agent.RingMethod, permission agent.Permission, tags []uuid.UUID, addresses []cmaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AgentCreate",
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

		TMCreate: getCurTime(),
		TMUpdate: defaultTimeStamp,
		TMDelete: defaultTimeStamp,
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

// AgentDelete updates the agent's basic info.
func (h *agentHandler) AgentDelete(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentDelete",
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

// AgentLogin validate the username and password.
func (h *agentHandler) AgentLogin(ctx context.Context, customerID uuid.UUID, username, password string) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AgentLogin",
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

// AgentUpdateBasicInfo updates the agent's basic info.
func (h *agentHandler) AgentUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AgentUpdateBasicInfo",
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

// AgentUpdatePassword updates the agent's password.
func (h *agentHandler) AgentUpdatePassword(ctx context.Context, id uuid.UUID, password string) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdatePassword",
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

// AgentUpdatePermission updates the agent's permission.
func (h *agentHandler) AgentUpdatePermission(ctx context.Context, id uuid.UUID, permission agent.Permission) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdatePermission",
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

// AgentUpdateTagIDs updates the agent's tags.
func (h *agentHandler) AgentUpdateTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdateTagIDs",
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

// AgentUpdateAddresses updates the agent's addresses.
func (h *agentHandler) AgentUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []cmaddress.Address) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdateAddresses",
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

// AgentUpdateStatus updates the agent's status.
func (h *agentHandler) AgentUpdateStatus(ctx context.Context, id uuid.UUID, status agent.Status) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdateStatus",
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

// AgentDial dials to the agent.
func (h *agentHandler) AgentDial(ctx context.Context, id uuid.UUID, source *cmaddress.Address, flowID, masterCallID uuid.UUID) (*agentdial.AgentDial, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AgentDial",
		"agent_id":       id,
		"flow_id":        flowID,
		"master_call_id": masterCallID,
	})
	log.Debug("Dialing to the agent.")

	// get agent
	ag, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}
	log.WithField("agent", ag).Debug("Found agent.")

	// check agent's status and addresses
	if ag.Status != agent.StatusAvailable {
		log.Debugf("Agent is not available. status: %s", ag.Status)
		return nil, fmt.Errorf("agant is not available")
	} else if len(ag.Addresses) == 0 {
		log.Debugf("Agent has no address.")
		return nil, fmt.Errorf("agent has no address")
	}

	// set agent status to ringing
	if err := h.db.AgentSetStatus(ctx, ag.ID, agent.StatusRinging); err != nil {
		log.Errorf("Could not update the agent's status. err: %v", err)
		return nil, err
	}

	// generate the call ids and agentcall info
	agentDialID := uuid.Must(uuid.NewV4())
	callIDs := []uuid.UUID{}
	for i := 0; i < len(ag.Addresses); i++ {
		agentCallID := uuid.Must(uuid.NewV4())
		callIDs = append(callIDs, agentCallID)

		ac := &agentcall.AgentCall{
			ID:          agentCallID,
			CustomerID:  ag.CustomerID,
			AgentID:     ag.ID,
			AgentDialID: agentDialID,
		}
		if err := h.db.AgentCallCreate(ctx, ac); err != nil {
			log.Errorf("Could not create a agent call. err: %v", err)
		}
	}
	log.Debugf("call id info. call_ids: %v", callIDs)

	// create agentdial
	ad := &agentdial.AgentDial{
		ID:           agentDialID,
		CustomerID:   ag.CustomerID,
		AgentID:      ag.ID,
		AgentCallIDs: callIDs,
	}
	if err := h.db.AgentDialCreate(ctx, ad); err != nil {
		log.Errorf("Could not create an agent dial. err: %v", err)
	}

	if ag.RingMethod == agent.RingMethodLinear {
		log.Errorf("Currently, support the ringall only.")
		return nil, fmt.Errorf("unsupport ringmethod")
	}

	// dial
	for i, address := range ag.Addresses {
		c, err := h.reqHandler.CMV1CallCreateWithID(ctx, callIDs[i], ag.CustomerID, flowID, masterCallID, source, &address)
		if err != nil {
			log.Errorf("Could not create a call. err: %v", err)
			continue
		}
		log.WithField("call", c).Debug("Created a call")
	}

	return ad, nil
}
