package queuehandler

import (
	"context"

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

	// get filters
	filters := map[amagent.Field]any{
		amagent.FieldDeleted:    false,
		amagent.FieldCustomerID: q.CustomerID.String(),
	}
	// omit the key entirely when the queue has no tags, rather than sending
	// an empty string -- an untagged queue means "no tag constraint, route
	// to any available agent", and the key's presence/absence is what
	// downstream layers use to distinguish that from an explicit (and
	// therefore validated) tag filter.
	if tagIDs := amagent.FormatTagIDsFilter(q.TagIDs); tagIDs != "" {
		filters[amagent.FieldTagIDs] = tagIDs
	}
	if status != amagent.StatusNone {
		filters[amagent.FieldStatus] = string(status)
	}

	// get agents
	res, err := h.reqHandler.AgentV1AgentList(ctx, h.utilHandler.TimeGetCurTime(), 100, filters)
	if err != nil {
		log.Errorf("Could not get agents. err: %v", err)
		return nil, err
	}

	return res, nil
}
