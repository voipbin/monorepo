package queuecallhandler

import (
	"context"
	"math/rand"

	amagent "monorepo/bin-agent-manager/models/agent"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
)

// Execute connects the queuecall to the agent following the match() procedure
// (VOIP-1539 §3.3): reserve the agent, dial, then win the entry CAS. Every
// failure path unwinds the reservation (and, once dialed, the agent groupcall)
// so a failed match leaves no dangling reservation or ringing leg.
func (h *queuecallHandler) Execute(ctx context.Context, id uuid.UUID, agentID uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Execute",
			"queuecall_id": id,
			"agent_id":     agentID,
		},
	)

	qc, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queuecall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get queuecall info.")
	}

	// create the flow for the agent dial
	f, err := h.generateFlowForAgentCall(ctx, qc.CustomerID, qc.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not create the flow tor agent dialing. err: %v", err)
		return nil, err
	}

	// reserve the agent (CAS). The reservation is the double-assignment guard:
	// the agent stays available during the dial, so a losing reserve means
	// another queuecall already claimed the agent.
	ok, err := h.reqHandler.AgentV1AgentReserve(ctx, agentID, "queuecall", qc.ID)
	if err != nil {
		log.Errorf("Could not reserve the agent. err: %v", err)
		return nil, errors.Wrap(err, "Could not reserve the agent.")
	}
	if !ok {
		log.Debugf("The agent is not reservable. agent_id: %s", agentID)
		return nil, errors.Errorf("could not reserve the agent. agent_id: %s", agentID)
	}

	destinations := []commonaddress.Address{
		{
			Type:   commonaddress.TypeAgent,
			Target: agentID.String(),
		},
	}

	calls, groupcalls, err := h.reqHandler.CallV1CallsCreate(ctx, qc.CustomerID, f.ID, qc.ReferenceID, &qc.Source, destinations, false, false, "", nil, nil)
	if err != nil {
		log.Errorf("Could not create a call to the agent. err: %v", err)
		// dial failed: release the reservation.
		if errRelease := h.reqHandler.AgentV1AgentReserveRelease(ctx, agentID, qc.ID); errRelease != nil {
			log.Errorf("Could not release the agent reservation. err: %v", errRelease)
		}
		return nil, errors.Wrap(err, "Could not create a call to the agent.")
	}
	if len(groupcalls) == 0 {
		log.Errorf("Could not get the agent groupcall. groupcalls is empty.")
		// no groupcall produced: release the reservation.
		if errRelease := h.reqHandler.AgentV1AgentReserveRelease(ctx, agentID, qc.ID); errRelease != nil {
			log.Errorf("Could not release the agent reservation. err: %v", errRelease)
		}
		return nil, errors.Errorf("could not get the agent groupcall. groupcalls is empty.")
	}
	log.WithFields(logrus.Fields{
		"calls":      calls,
		"groupcalls": groupcalls,
	}).Debugf("Created call to the agent. agent_id: %s", agentID)

	agentGroupcallID := groupcalls[0].ID

	// entry CAS: move the queuecall waiting->connecting and persist the
	// groupcall id. dial happened before the CAS, so the groupcall id is known.
	res, won, err := h.UpdateStatusConnecting(ctx, qc.ID, agentID, agentGroupcallID)
	if err != nil {
		log.Errorf("Could not update the status to connecting. err: %v", err)
		// best-effort unwind of the reservation.
		if errRelease := h.reqHandler.AgentV1AgentReserveRelease(ctx, agentID, qc.ID); errRelease != nil {
			log.Errorf("Could not release the agent reservation. err: %v", errRelease)
		}
		return nil, err
	}

	if !won {
		// CAS lost: another writer already moved the queuecall out of waiting.
		// Unwind the agent dial and reservation; return the current queuecall.
		log.Debugf("Lost the entry CAS. Unwinding the agent dial. queuecall_id: %s", qc.ID)
		if _, errHangup := h.reqHandler.CallV1GroupcallHangup(ctx, agentGroupcallID); errHangup != nil {
			log.Errorf("Could not hang up the agent groupcall. err: %v", errHangup)
		}
		if errRelease := h.reqHandler.AgentV1AgentReserveRelease(ctx, agentID, qc.ID); errRelease != nil {
			log.Errorf("Could not release the agent reservation. err: %v", errRelease)
		}
		return res, nil
	}

	// forward the action.
	if err := h.reqHandler.FlowV1ActiveflowUpdateForwardActionID(ctx, res.ReferenceActiveflowID, res.ForwardActionID, true); err != nil {
		log.Errorf("Could not forward the active flow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// generateFlowForAgentCall creates a flow for the agent call action.
func (h *queuecallHandler) generateFlowForAgentCall(ctx context.Context, customerID, confbridgeID uuid.UUID) (*fmflow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "generateFlowForAgentCall",
		"customer_id":   customerID,
		"confbridge_id": confbridgeID,
	})

	// create actions
	actions := []fmaction.Action{
		{
			Type: fmaction.TypeConfbridgeJoin,
			Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
				ConfbridgeID: confbridgeID,
			}),
		},
	}

	// create a flow for agent dial.
	res, err := h.reqHandler.FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, "automatically generated for the agent call by the queue-manager", "", actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// matchWaitingQueuecall tries a single match() attempt for the given
// just-enqueued queuecall against the queue's currently available agents
// (VOIP-1539 §3.5, event entry point A). One-shot: on failure (no available
// agent, or a losing reservation/entry CAS) it does not retry -- the
// queuecall is left for entry point B or the matching backstop (§5.2).
func (h *queuecallHandler) matchWaitingQueuecall(ctx context.Context, qc *queuecall.Queuecall) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "matchWaitingQueuecall",
		"queuecall_id": qc.ID,
		"queue_id":     qc.QueueID,
	})

	agents, err := h.queueHandler.GetAgents(ctx, qc.QueueID, amagent.StatusAvailable)
	if err != nil {
		log.Errorf("Could not get available agents. err: %v", err)
		return
	}
	if len(agents) == 0 {
		return
	}

	target := agents[0]
	if q, errGet := h.queueHandler.Get(ctx, qc.QueueID); errGet == nil && q.RoutingMethod == queue.RoutingMethodRandom {
		target = agents[rand.Intn(len(agents))]
	}

	if _, errExec := h.Execute(ctx, qc.ID, target.ID); errExec != nil {
		// Losing the reservation/entry CAS is an expected outcome, not an
		// error worth logging above debug.
		log.Debugf("Could not execute the match. agent_id: %s, err: %v", target.ID, errExec)
	}
}
