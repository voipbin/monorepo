package queuecallhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Execute connects the queuecall to the agent.
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

	// create the flow for the agnet dial
	f, err := h.generateFlowForAgentCall(ctx, qc.CustomerID, qc.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not create the flow tor agent dialing. err: %v", err)
		return nil, err
	}

	// dial to the agent
	log.WithField("agent_id", agentID).Debugf("Dialing the agent. agent_id: %s", agentID)
	agentDial, err := h.reqHandler.AgentV1AgentDial(ctx, agentID, &qc.Source, f.ID, qc.ReferenceID)
	if err != nil {
		log.Errorf("Could not dial to the agent. Send the request again with 1 sec delay. err: %v", err)
		return nil, err
	}
	log.WithField("agent_dial", agentDial).Debugf("Created agent dial. agent_dial_id: %s", agentDial.ID)

	// update the queuecall status to connecting
	log.Debugf("Update the queuecall status to connecting. agent_id: %s", agentID)
	res, err := h.UpdateStatusConnecting(ctx, qc.ID, agentID)
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
	res, err := h.reqHandler.FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, "automatically generated for the agent call by the queue-manager", "", actions, false)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", res).Debug("Created a flow.")

	return res, nil
}
