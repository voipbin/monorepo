package queuehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// GetAgents retruns list of agents of the given queue and status
func (h *queueHandler) GetAgents(ctx context.Context, id uuid.UUID, status amagent.Status) ([]amagent.Agent, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "GetAgents",
			"id":   id,
		},
	)

	q, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	// get agents
	switch status {
	case amagent.StatusNone:
		res, err := h.reqHandler.AgentV1AgentGetsByTagIDs(ctx, q.CustomerID, q.TagIDs)
		if err != nil {
			log.Errorf("Could not get agents. err: %v", err)
			return nil, err
		}

		return res, nil

	default:
		res, err := h.reqHandler.AgentV1AgentGetsByTagIDsAndStatus(ctx, q.CustomerID, q.TagIDs, status)
		if err != nil {
			log.Errorf("Could not get agents. err: %v", err)
			return nil, err
		}

		return res, nil
	}
}
