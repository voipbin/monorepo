package queuehandler

import (
	"context"
	"strings"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// GetAgents retruns list of agents of the given queue and status
func (h *queueHandler) GetAgents(ctx context.Context, id uuid.UUID, status amagent.Status) ([]amagent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "GetAgents",
		"id":     id,
		"status": status,
	})

	q, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	// get tag ids
	tmpIDs := []string{}
	for _, id := range q.TagIDs {
		tmpIDs = append(tmpIDs, id.String())
	}
	tagIds := strings.Join(tmpIDs, ",")

	// get filters
	filters := map[amagent.Field]any{
		amagent.FieldDeleted:    false,
		amagent.FieldCustomerID: q.CustomerID.String(),
		amagent.FieldTagIDs:     tagIds,
	}
	if status != amagent.StatusNone {
		filters[amagent.FieldStatus] = string(status)
	}

	// get agents
	res, err := h.reqHandler.AgentV1AgentGets(ctx, h.utilHandler.TimeGetCurTime(), 100, filters)
	if err != nil {
		log.Errorf("Could not get agents. err: %v", err)
		return nil, err
	}

	return res, nil
}
