package queuehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// GetAgents retruns list of agents of the given queue and status
func (h *queueHandler) GetAgents(ctx context.Context, id uuid.UUID, status amagent.Status) ([]agent.Agent, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "SearchAgent",
			"id":   id,
		},
	)

	q, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	// get agents
	if status == amagent.StatusNone {
		res, err := h.reqHandler.AMV1AgentGetsByTagIDs(ctx, q.CustomerID, q.TagIDs)
		if err != nil {
			log.Errorf("Could not get agents. err: %v", err)
			return nil, err
		}

		return res, nil
	}

	res, err := h.reqHandler.AMV1AgentGetsByTagIDsAndStatus(ctx, q.CustomerID, q.TagIDs, status)
	if err != nil {
		log.Errorf("Could not get agents. err: %v", err)
		return nil, err
	}

	return res, nil
}
