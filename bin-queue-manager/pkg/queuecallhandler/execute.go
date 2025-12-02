package queuecallhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queuecall"
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

	destinations := []commonaddress.Address{
		{
			Type:   commonaddress.TypeAgent,
			Target: agentID.String(),
		},
	}

	calls, groupcalls, err := h.reqHandler.CallV1CallsCreate(ctx, qc.CustomerID, f.ID, qc.ReferenceID, &qc.Source, destinations, false, false)
	if err != nil {
		log.Errorf("Could not create a call to the agent. err: %v", err)
		return nil, errors.Wrap(err, "Could not create a call to the agent.")
	}
	log.WithFields(logrus.Fields{
		"calls":      calls,
		"groupcalls": groupcalls,
	}).Debugf("Created call to the agent. agent_id: %s", agentID)

	// update the queuecall status to connecting
	res, err := h.UpdateStatusConnecting(ctx, qc.ID, agentID)
	if err != nil {
		log.Errorf("Could not update the status to connecting. agent id. err: %v", err)
		return nil, err
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
