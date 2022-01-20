package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// agentGet validates the agent's ownership and returns the agent info.
func (h *serviceHandler) agentGet(ctx context.Context, u *user.User, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "agentGet",
			"user_id":  u.ID,
			"agent_id": id,
		},
	)

	// send request
	tmp, err := h.reqHandler.AMV1AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the agent info. err: %v", err)
		return nil, err
	}
	log.WithField("agent", tmp).Debug("Received result.")

	if u.Permission != user.PermissionAdmin && u.ID != tmp.UserID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := agent.ConvertToAgent(tmp)
	return res, nil
}

// AgentCreate sends a request to agent-manager
// to creating an agent.
// it returns created agent info if it succeed.
func (h *serviceHandler) AgentCreate(
	u *user.User,
	username string,
	password string,
	name string,
	detail string,
	webhookMethod string,
	webhookURI string,
	ringMethod string,
	permission uint64,
	tagIDs []uuid.UUID,
	addresses []address.Address,
) (*agent.Agent, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentCreate",
		"user":     u.ID,
		"username": u.Username,
	})

	// convert addresses
	cmAddresses := []cmaddress.Address{}
	for _, addr := range addresses {
		cmAddresses = append(cmAddresses, *address.ConvertToCMAddress(&addr))
	}

	// send request
	log.Debug("Creating a new agent.")
	tmp, err := h.reqHandler.AMV1AgentCreate(ctx, 30, u.ID, username, password, name, detail, webhookMethod, webhookURI, amagent.RingMethod(ringMethod), amagent.Permission(permission), tagIDs, cmAddresses)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}
	log.WithField("agent", tmp).Debug("Received result.")

	// create result
	res := agent.ConvertToAgent(tmp)

	return res, nil
}

// AgentGet sends a request to agent-manager
// to getting an agent.
func (h *serviceHandler) AgentGet(u *user.User, agentID uuid.UUID) (*agent.Agent, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentGet",
		"user_id":  u.ID,
		"username": u.Username,
		"agent_id": agentID,
	})

	res, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// AgentGet sends a request to agent-manager
// to getting a list of agents.
// it returns agent info if it succeed.
func (h *serviceHandler) AgentGets(u *user.User, size uint64, token string, tagIDs []uuid.UUID, status agent.Status) ([]*agent.Agent, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentGets",
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})

	if token == "" {
		token = getCurTime()
	}

	// get agents
	var tmps []amagent.Agent
	var err error
	if len(tagIDs) > 0 && status != agent.StatusNone {
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
	res := []*agent.Agent{}
	for _, tmp := range tmps {
		c := agent.ConvertToAgent(&tmp)
		res = append(res, c)
	}

	return res, nil
}

// AgentDelete sends a request to call-manager
// to delete the agent.
func (h *serviceHandler) AgentDelete(u *user.User, agentID uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentDelete",
		"user":     u.ID,
		"username": u.Username,
		"agent_id": agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return err
	}

	// send request
	if err := h.reqHandler.AMV1AgentDelete(ctx, agentID); err != nil {
		log.Infof("Could not delete the agent info. err: %v", err)
		return err
	}

	return nil
}

// AgentDelete sends a request to call-manager
// to delete the agent.
func (h *serviceHandler) AgentLogin(userID uint64, username, password string) (string, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentLogin",
		"user_id":  userID,
		"username": username,
		"password": len(password),
	})

	// send request
	ag, err := h.reqHandler.AMV1AgentLogin(ctx, 30, userID, username, password)
	if err != nil {
		log.Warningf("Could not agent login. err: %v", err)
		return "", err
	}
	tmp := agent.ConvertToAgent(ag)

	serialized := tmp.Serialize()
	token, err := middleware.GenerateToken(serialized)
	if err != nil {
		logrus.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return token, nil
}

// AgentUpdate sends a request to agent-manager
// to update the agent info.
func (h *serviceHandler) AgentUpdate(u *user.User, agentID uuid.UUID, name, detail string, ringMethod agent.RingMethod) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdate",
		"user":     u.ID,
		"username": u.Username,
		"agent_id": agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return err
	}

	// send request
	if err := h.reqHandler.AMV1AgentUpdate(ctx, agentID, name, detail, string(ringMethod)); err != nil {
		log.Infof("Could not delete the agent info. err: %v", err)
		return err
	}

	return nil
}

// AgentUpdate sends a request to agent-manager
// to update the agent's addresses info.
func (h *serviceHandler) AgentUpdateAddresses(u *user.User, agentID uuid.UUID, addresses []address.Address) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdateAddresses",
		"user":     u.ID,
		"username": u.Username,
		"agent_id": agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return err
	}

	addrs := []cmaddress.Address{}
	for _, tmp := range addresses {
		addr := address.ConvertToCMAddress(&tmp)
		addrs = append(addrs, *addr)
	}

	// send request
	if err := h.reqHandler.AMV1AgentUpdateAddresses(ctx, agentID, addrs); err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return err
	}

	return nil
}

// AgentUpdateTagIDs sends a request to agent-manager
// to update the agent's tag_ids info.
func (h *serviceHandler) AgentUpdateTagIDs(u *user.User, agentID uuid.UUID, tagIDs []uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdateTagIDs",
		"user":     u.ID,
		"username": u.Username,
		"agent_id": agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return err
	}

	// send request
	if err := h.reqHandler.AMV1AgentUpdateTagIDs(ctx, agentID, tagIDs); err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return err
	}

	return nil
}

// AgentUpdateStatus sends a request to agent-manager
// to update the agent status info.
func (h *serviceHandler) AgentUpdateStatus(u *user.User, agentID uuid.UUID, status agent.Status) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "AgentUpdateStatus",
		"user":     u.ID,
		"username": u.Username,
		"agent_id": agentID,
	})

	_, err := h.agentGet(ctx, u, agentID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return err
	}

	// send request
	if err := h.reqHandler.AMV1AgentUpdateStatus(ctx, agentID, amagent.Status(status)); err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return err
	}

	return nil
}
