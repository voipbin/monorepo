package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/sirupsen/logrus"
)

// ServiceAgentMeGet retrieves the given authenticated agent's information.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeGet(ctx context.Context, a *amagent.Agent) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentMeGet",
		"agent": a,
	})

	tmp, err := h.agentGet(ctx, a, a.ID)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentMeUpdate updates the authenticated agent's details.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeUpdate(ctx context.Context, a *amagent.Agent, name string, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentMeUpdate",
		"agent": a,
	})

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdate(ctx, a.ID, name, detail, ringMethod)
	if err != nil {
		log.Infof("Could not update the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentMeUpdateAddresses updates the authenticated agent's address information.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeUpdateAddresses(ctx context.Context, a *amagent.Agent, addresses []commonaddress.Address) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentMeUpdateAddresses",
		"agent": a,
	})

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdateAddresses(ctx, a.ID, addresses)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentMeUpdateStatus updates the authenticated agent's status information.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeUpdateStatus(ctx context.Context, a *amagent.Agent, status amagent.Status) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentMeUpdateStatus",
		"agent": a,
	})

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdateStatus(ctx, a.ID, status)
	if err != nil {
		log.Infof("Could not update the agent addresses. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentMeUpdatePassword updates the authenticated agent's password.
// It returns updated agent info.
func (h *serviceHandler) ServiceAgentMeUpdatePassword(ctx context.Context, a *amagent.Agent, password string) (*amagent.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "ServiceAgentMeUpdatePassword",
		"agent":    a,
		"password": len(password),
	})

	// send request
	tmp, err := h.reqHandler.AgentV1AgentUpdatePassword(ctx, 30000, a.ID, password)
	if err != nil {
		log.Infof("Could not update the agent password. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
