package callhandler

import (
	"context"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

// setVariables sets the variables
func (h *callHandler) setVariables(ctx context.Context, c *call.Call) error {

	variables := map[string]string{

		// source
		"voipbin.call.source.name":        c.Source.Name,
		"voipbin.call.source.detail":      c.Source.Detail,
		"voipbin.call.source.target":      c.Source.Target,
		"voipbin.call.source.target_name": c.Source.Target,
		"voipbin.call.source.type":        string(c.Source.Type),

		// destination
		"voipbin.call.destination.name":        c.Destination.Name,
		"voipbin.call.destination.detail":      c.Destination.Detail,
		"voipbin.call.destination.target":      c.Destination.Target,
		"voipbin.call.destination.target_name": c.Destination.TargetName,
		"voipbin.call.destination.type":        string(c.Destination.Type),

		// others
		"voipbin.call.direction":      string(c.Direction),
		"voipbin.call.master_call_id": c.MasterCallID.String(),
	}

	for key, val := range variables {
		if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, key, val); err != nil {
			return fmt.Errorf("could not set the variable. key: %s, val: %s, err: %v", key, val, err)
		}
	}

	return nil
}
