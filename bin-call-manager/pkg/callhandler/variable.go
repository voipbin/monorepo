package callhandler

import (
	"context"
	"fmt"

	"monorepo/bin-call-manager/models/call"
)

// setVariablesCall sets the variables
func (h *callHandler) setVariablesCall(ctx context.Context, c *call.Call) error {

	variables := map[string]string{
		variableCallID: c.ID.String(),

		// source
		variableCallSourceName:       c.Source.Name,
		variableCallSourceDetail:     c.Source.Detail,
		variableCallSourceTarget:     c.Source.Target,
		variableCallSourceTargetName: c.Source.TargetName,
		variableCallSourceType:       string(c.Source.Type),

		// destination
		variableCallDestinationName:       c.Destination.Name,
		variableCallDestinationDetail:     c.Destination.Detail,
		variableCallDestinationTarget:     c.Destination.Target,
		variableCallDestinationTargetName: c.Destination.TargetName,
		variableCallDestinationType:       string(c.Destination.Type),

		// others
		variableCallDirection:    string(c.Direction),
		variableCallMasterCallID: c.MasterCallID.String(),
		variableCallDigits:       "",
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, c.ActiveFlowID, variables); errSet != nil {
		return fmt.Errorf("could not set the variable. variables: %s, err: %v", variables, errSet)
	}

	return nil
}
