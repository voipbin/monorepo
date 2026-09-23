package queuecallhandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queuecall"
)

// EventCallCallHangup handles call-manager call_hungup
func (h *queuecallHandler) EventCallCallHangup(ctx context.Context, referenceID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "EventCallCallHangup",
		"reference_id": referenceID,
	})

	qc, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		// no queuecall exist. nothing to do
		return
	}

	if qc.TMEnd != nil || qc.Status == queuecall.StatusService {
		// already done or other handler will deal with it.
		// nothing to do.
		return
	}

	_, err = h.UpdateStatusAbandoned(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status abandoned.")
	}
}

// EventCallConfbridgeJoined handles call-manager confbridge_join
func (h *queuecallHandler) EventCallConfbridgeJoined(ctx context.Context, referenceID uuid.UUID, confbridgeID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "EventCallConfbridgeJoined",
		"reference_id":  referenceID,
		"confbridge_id": confbridgeID,
	})

	qc, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		// no queuecall exist. nothing to do
		return
	}

	if qc.TMEnd != nil || qc.ConfbridgeID != confbridgeID {
		// already done or other handler will deal with it.
		// nothing to do.
		return
	}

	// update queuecall info
	res, err := h.UpdateStatusService(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status to service. err: %v", err)
		return
	}
	log.WithField("queuecall", res).Debugf("Updated queuecall status service. queuecall_id: %s", res.ID)
}

// EventCallConfbridgeLeaved handles call-manager confbridge_leaved
func (h *queuecallHandler) EventCallConfbridgeLeaved(ctx context.Context, referenceID uuid.UUID, confbridgeID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "EventCallConfbridgeLeaved",
		"reference_id":  referenceID,
		"confbridge_id": confbridgeID,
	})

	// get queuecall
	qc, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		return
	}

	// validate the queuecall info
	if qc.ConfbridgeID != confbridgeID || qc.TMEnd != nil {
		// queuecall is not valid.
		return
	}

	// update queuecall status to done
	res, err := h.UpdateStatusDone(ctx, qc)
	if err != nil {
		log.Errorf("Could not update the queuecall status to done. err: %v", err)
		return
	}
	log.WithField("queuecall", res).Debugf("Updated queuecall status done. queuecall_id: %s", res.ID)
}

// EventAMAgentAvailable handles the agent-manager's agent_status_updated
// event, filtered to status==available by the caller (VOIP-1539 §3.5, event
// entry point B). It reverse-looks-up the queues this agent is eligible for
// (queuehandler.GetQueuesByAgent, the OR/intersection-tag symmetric
// counterpart of GetAgents), then tries match() once against the single
// oldest-waiting queuecall in each of those queues (FIFO, tm_create ASC).
// One available agent can win at most one dial -- the agent reservation CAS
// inside Execute is what actually enforces that; this loop just offers each
// eligible queue a shot. Any queuecall this loop doesn't reach (agent
// reservation already lost to another queue's match, or 0 waiting in that
// queue) is left for the next entry-point trigger or the matching backstop
// (§5.2) -- entry points are a best-effort optimization, not the only path to
// a match.
func (h *queuecallHandler) EventAMAgentAvailable(ctx context.Context, agent amagent.Agent) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventAMAgentAvailable",
		"agent_id": agent.ID,
	})

	qs, err := h.queueHandler.GetQueuesByAgent(ctx, agent)
	if err != nil {
		log.Errorf("Could not get queues for the agent. err: %v", err)
		return
	}

	for _, q := range qs {
		qcs, err := h.db.QueuecallListOldestWaiting(ctx, q.ID, 1)
		if err != nil {
			log.Errorf("Could not get the oldest waiting queuecall. queue_id: %s, err: %v", q.ID, err)
			continue
		}
		if len(qcs) == 0 {
			continue
		}

		if _, err := h.Execute(ctx, qcs[0].ID, agent.ID); err != nil {
			// Losing the reservation/entry CAS is an expected outcome when
			// multiple queues raced for the same agent, or a competing entry
			// point already claimed this queuecall -- not an error worth
			// logging above debug.
			log.Debugf("Could not execute the match. queue_id: %s, queuecall_id: %s, err: %v", q.ID, qcs[0].ID, err)
		}
	}
}

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *queuecallHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCUCustomerDeleted",
		"customer_id": cu.ID,
	})
	log.Debugf("Deleting all queues in customer. customer_id: %s", cu.ID)

	// get all queuecalls in customer
	filters := map[queuecall.Field]any{
		queuecall.FieldCustomerID: cu.ID,
		queuecall.FieldDeleted:    false,
	}
	qs, err := h.List(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets queuecalls list. err: %v", err)
		return errors.Wrap(err, "could not get queuecalls list")
	}

	// kick all queuecalls
	for _, q := range qs {
		log.Debugf("Kicking out the queuecalls from the queue. queuecall_id: %s", q.ID)
		qc, err := h.kickForce(ctx, q.ID)
		if err != nil {
			log.Errorf("Could not kick out the queuecall from the queue. err: %v", err)
			continue
		}
		log.WithField("queuecall", qc).Debugf("Kicked out the queuecall. queuecall_id: %s", qc.ID)
	}

	// delete all queuecalls
	for _, q := range qs {
		log.Debugf("Deleting queuecall info. queuecall_id: %s", q.ID)
		tmp, err := h.Delete(ctx, q.ID)
		if err != nil {
			log.Errorf("Could not delete queuecall info. err: %v", err)
			continue
		}
		log.WithField("queuecall", tmp).Debugf("Deleted queuecall info. queuecall_id: %s", tmp.ID)
	}

	return nil
}
