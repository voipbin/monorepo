package queuehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
)

// queueListPageSize is the page size GetQueuesByAgent uses while paging
// through QueueList. Kept modest relative to queueListMaxPages so a
// pathological customer (many queues) shows up as extra round trips rather
// than a single unbounded query.
const queueListPageSize = 100

// queueListMaxPages caps the number of pages GetQueuesByAgent will walk for a
// single customer, so a data anomaly (e.g. a runaway queue-creation bug)
// degrades to "some queues missed" instead of an unbounded loop. Sized well
// above any expected customer's queue count (VOIP-1539 §3.5: real customers
// have a small number of queues; this is a safety bound, not a tuned limit).
const queueListMaxPages = 1000

// hasCommonTagID reports whether a and b share at least one element. Mirrors
// the OR/intersection semantics of agent-manager's applyTagIDsFilter
// (bin-agent-manager/pkg/dbhandler/agent.go): "has any matching skill", not
// "has all of them".
func hasCommonTagID(a []uuid.UUID, b []uuid.UUID) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := make(map[uuid.UUID]struct{}, len(b))
	for _, id := range b {
		set[id] = struct{}{}
	}
	for _, id := range a {
		if _, ok := set[id]; ok {
			return true
		}
	}
	return false
}

// GetQueuesByAgent returns the queues the given agent is eligible to serve
// (VOIP-1539 §3.5, event entry point B): the reverse of GetAgents. A queue is
// a match when it is untagged (no tag constraint -- always included) or when
// queue.TagIDs and agent.TagIDs intersect (queue.TagIDs ∩ agent.TagIDs ≠ ∅),
// the same OR semantics agent-manager's applyTagIDsFilter uses for the
// forward direction, so a queue that would accept this agent via GetAgents
// is never missed here.
//
// QueueList's tag_ids is a JSON column filter that can't express a
// set-intersection predicate in SQL, so this walks QueueList(CustomerID,
// deleted=false) pages and applies the intersection check in the
// application layer. Pages are walked (token = last row's tm_create) until a
// short page (or queueListMaxPages) is reached, so a customer with more
// queues than one page still gets full coverage instead of silently missing
// the tail.
func (h *queueHandler) GetQueuesByAgent(ctx context.Context, agent amagent.Agent) ([]*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "GetQueuesByAgent",
		"agent_id": agent.ID,
	})

	filters := map[queue.Field]any{
		queue.FieldDeleted:    false,
		queue.FieldCustomerID: agent.CustomerID.String(),
	}

	res := []*queue.Queue{}
	token := ""
	for page := 0; page < queueListMaxPages; page++ {
		qs, err := h.db.QueueList(ctx, queueListPageSize, token, filters)
		if err != nil {
			log.Errorf("Could not get queues. err: %v", err)
			return nil, err
		}

		for _, q := range qs {
			if len(q.TagIDs) == 0 || hasCommonTagID(q.TagIDs, agent.TagIDs) {
				res = append(res, q)
			}
		}

		if uint64(len(qs)) < queueListPageSize {
			break
		}

		last := qs[len(qs)-1]
		if last.TMCreate == nil {
			break
		}
		token = last.TMCreate.UTC().Format(utilhandler.ISO8601Layout)
	}

	return res, nil
}

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
