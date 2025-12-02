package activeflowhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
)

// generateFlowForAgentCall creates a flow for the agent call action.
func (h *activeflowHandler) generateFlowForAgentCall(ctx context.Context, customerID, confbridgeID uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "generateFlowForAgentCall",
		"confbridge_id": confbridgeID,
	})

	// create actions
	actions := []action.Action{
		{
			Type: action.TypeConfbridgeJoin,
			Option: action.ConvertOption(action.OptionConfbridgeJoin{
				ConfbridgeID: confbridgeID,
			}),
		},
	}

	// create a flow for agent dial.
	res, err := h.reqHandler.FlowV1FlowCreate(ctx, customerID, flow.TypeFlow, "automatically generated for the agent call", "", actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", res).Debug("Created a flow.")

	return res, nil
}
