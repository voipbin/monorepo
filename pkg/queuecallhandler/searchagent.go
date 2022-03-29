package queuecallhandler

import (
	"context"
	"encoding/json"
	"math/rand"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

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
		if err := h.reqHandler.FMV1ActiveflowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ExitActionID, true); err != nil {
			log.Errorf("Could not forward the call. err: %v", err)
		}
		return
	}

	// create the flow for the agnet dial
	f, err := h.generateFlowForAgentCall(ctx, qc.CustomerID, qc.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not create the flow tor agent dialing. err: %v", err)
		_ = h.reqHandler.QMV1QueuecallSearchAgent(ctx, qc.ID, defaultDelaySearchAgent)
		return
	}

	// dial to the agent
	log.WithField("agent", targetAgent).Debugf("Dialing the agent. agent_id: %s", targetAgent.ID)
	agentDial, err := h.reqHandler.AMV1AgentDial(ctx, targetAgent.ID, &qc.Source, f.ID, qc.ReferenceID)
	if err != nil {
		log.Errorf("Could not dial to the agent. Send the request again with 1 sec delay. err: %v", err)
		_ = h.reqHandler.QMV1QueuecallSearchAgent(ctx, qc.ID, defaultDelaySearchAgent)
		return
	}
	log.WithField("agent_dial", agentDial).Debugf("Created agent dial. agent_dial_id: %s", agentDial.ID)

	// forward the action.
	log.Debugf("Setting the forward action id. forward_action_id: %s", qc.ForwardActionID)
	if err := h.reqHandler.FMV1ActiveflowUpdateForwardActionID(ctx, qc.ReferenceActiveflowID, qc.ForwardActionID, true); err != nil {
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

// generateFlowForAgentCall creates a flow for the agent call action.
func (h *queuecallHandler) generateFlowForAgentCall(ctx context.Context, customerID, confbridgeID uuid.UUID) (*fmflow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "generateFlowForAgentCall",
		"confbridge_id": confbridgeID,
	})

	opt, err := json.Marshal(fmaction.OptionConfbridgeJoin{
		ConfbridgeID: confbridgeID,
	})
	if err != nil {
		log.Errorf("Could not marshal the action. err: %v", err)
		return nil, err
	}

	// create actions
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConfbridgeJoin,
			Option: opt,
		},
	}

	// create a flow for agent dial.
	res, err := h.reqHandler.FMV1FlowCreate(ctx, customerID, fmflow.TypeFlow, "automatically generated for the agent call by the queue-manager", "", actions, false)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", res).Debug("Created a flow.")

	return res, nil
}
