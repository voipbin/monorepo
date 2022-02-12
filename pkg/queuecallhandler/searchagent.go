package queuecallhandler

import (
	"context"
	"math/rand"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// SearchAgent search the available agent and dial to them.
func (h *queuecallHandler) SearchAgent(ctx context.Context, queuecallID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "SearchAgent",
			"queuecall_id": queuecallID,
		},
	)

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. Send the request again with 1 sec delay. err: %v", err)
		_ = h.reqHandler.QMV1QueuecallSearchAgent(ctx, queuecallID, defaultDelaySearchAgent)
		return
	}
	log = log.WithField("reference_id", qc.ReferenceID)
	log.WithField("queuecall", qc).Debug("Found queuecall info.")

	// check the status
	if qc.Status != queuecall.StatusWait {
		// no need to execute anymore.
		log.Errorf("The queuecall status is not wait. Will handle in other place. status: %v", qc.Status)
		return
	}

	// get available agents
	agents, err := h.reqHandler.AMV1AgentGetsByTagIDsAndStatus(ctx, qc.CustomerID, qc.TagIDs, amagent.StatusAvailable)
	if err != nil {
		log.Errorf("Could not get available agents. Send the request again with 1 sec delay. err: %v", err)
		_ = h.reqHandler.QMV1QueuecallSearchAgent(ctx, qc.ID, defaultDelaySearchAgent)
		return
	} else if len(agents) == 0 {
		log.Info("No available agent now. Send the request again with 1 sec delay.")
		_ = h.reqHandler.QMV1QueuecallSearchAgent(ctx, qc.ID, defaultDelaySearchAgent)
		return
	}

	targetAgent := amagent.Agent{}
	switch qc.RoutingMethod {
	case queue.RoutingMethodRandom:
		targetAgent = agents[rand.Intn(len(agents))]

	default:
		log.Errorf("Unsupported routing method. Exit from the queue. routing_method: %s", qc.RoutingMethod)
		if err := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ExitActionID, true); err != nil {
			log.Errorf("Could not forward the call. err: %v", err)
		}
		return
	}

	// dial to the agent
	log.WithField("agent", targetAgent).Debugf("Dialing the agent. agent_id: %s", targetAgent.ID)
	if err := h.reqHandler.AMV1AgentDial(ctx, targetAgent.ID, &qc.Source, qc.ConfbridgeID, qc.ReferenceID); err != nil {
		log.Errorf("Could not dial to the agent. Send the request again with 1 sec delay. err: %v", err)
		_ = h.reqHandler.QMV1QueuecallSearchAgent(ctx, qc.ID, defaultDelaySearchAgent)
		return
	}

	// forward the action.
	log.Debugf("Setting the forward action id. forward_action_id: %s", qc.ForwardActionID)
	if err := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ForwardActionID, true); err != nil {
		log.Errorf("Could not forward the active flow. err: %v", err)
		return
	}

	// update the queuecall
	log.Debugf("Update the queuecall service agent id. agent_id: %s", targetAgent.ID)
	if err := h.db.QueuecallSetServiceAgentID(ctx, qc.ID, targetAgent.ID); err != nil {
		log.Errorf("Could not ser the service agent id. err: %v", err)
		return
	}

	// get updated queuecall and notify
	tmp, err := h.db.QueuecallGet(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return
	}
	h.notifyhandler.PublishWebhookEvent(ctx, tmp.CustomerID, queuecall.EventTypeQueuecallEntering, tmp)
}
