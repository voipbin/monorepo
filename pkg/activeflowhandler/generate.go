package activeflowhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

// generateFlowForAgentCall creates a flow for the agent call action.
func (h *activeflowHandler) generateFlowForAgentCall(ctx context.Context, customerID, confbridgeID uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "generateFlowForAgentCall",
		"confbridge_id": confbridgeID,
	})

	opt, err := json.Marshal(action.OptionConfbridgeJoin{
		ConfbridgeID: confbridgeID,
	})
	if err != nil {
		log.Errorf("Could not marshal the action. err: %v", err)
		return nil, err
	}

	// create actions
	actions := []action.Action{
		{
			Type:   action.TypeConfbridgeJoin,
			Option: opt,
		},
	}

	// create a flow for agent dial.
	res, err := h.reqHandler.FlowV1FlowCreate(ctx, customerID, flow.TypeFlow, "automatically generated for the agent call", "", actions, false)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", res).Debug("Created a flow.")

	return res, nil
}
