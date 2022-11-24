package queuecallhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Execute connects the queuecall to the agent.
func (h *queuecallHandler) Execute(ctx context.Context, qc *queuecall.Queuecall, agent *amagent.Agent) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Execute",
			"queuecall_id": qc.ID,
			"agent_id":     agent.ID,
		},
	)

	// create the flow for the agnet dial
	f, err := h.generateFlowForAgentCall(ctx, qc.CustomerID, qc.ConferenceID)
	if err != nil {
		log.Errorf("Could not create the flow tor agent dialing. err: %v", err)
		return nil, err
	}

	// dial to the agent
	log.WithField("agent", agent).Debugf("Dialing the agent. agent_id: %s", agent.ID)
	agentDial, err := h.reqHandler.AgentV1AgentDial(ctx, agent.ID, &qc.Source, f.ID, qc.ReferenceID)
	if err != nil {
		log.Errorf("Could not dial to the agent. Send the request again with 1 sec delay. err: %v", err)
		return nil, err
	}
	log.WithField("agent_dial", agentDial).Debugf("Created agent dial. agent_dial_id: %s", agentDial.ID)

	// update the queuecall status to connecting
	log.Debugf("Update the queuecall status to connecting. agent_id: %s", agent.ID)
	res, err := h.UpdateStatusConnecting(ctx, qc.ID, agent.ID)
	if err != nil {
		log.Errorf("Could not update the status to connecting. agent id. err: %v", err)
		return nil, err
	}

	// forward the action.
	log.Debugf("Setting the forward action id. forward_action_id: %s", qc.ForwardActionID)
	if err := h.reqHandler.FlowV1ActiveflowUpdateForwardActionID(ctx, qc.ReferenceActiveflowID, qc.ForwardActionID, true); err != nil {
		log.Errorf("Could not forward the active flow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// generateFlowForAgentCall creates a flow for the agent call action.
func (h *queuecallHandler) generateFlowForAgentCall(ctx context.Context, customerID, conferenceID uuid.UUID) (*fmflow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "generateFlowForAgentCall",
		"conference_id": conferenceID,
	})

	opt, err := json.Marshal(fmaction.OptionConferenceJoin{
		ConferenceID: conferenceID,
	})
	if err != nil {
		log.Errorf("Could not marshal the action. err: %v", err)
		return nil, err
	}

	// create actions
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConferenceJoin,
			Option: opt,
		},
	}

	// create a flow for agent dial.
	res, err := h.reqHandler.FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, "automatically generated for the agent call by the queue-manager", "", actions, false)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", res).Debug("Created a flow.")

	return res, nil
}
