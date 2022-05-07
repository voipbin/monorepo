package callhandler

import (
	"context"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

// setVariables sets the variables
func (h *callHandler) setVariables(ctx context.Context, c *call.Call) error {

	// source
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.source.name", c.Source.Name); err != nil {
		return err
	}
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.source.detail", c.Source.Detail); err != nil {
		return err
	}
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.source.target", c.Source.Target); err != nil {
		return err
	}
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.source.target_name", c.Source.TargetName); err != nil {
		return err
	}
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.source.type", string(c.Source.Type)); err != nil {
		return err
	}

	// destination
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.destination.name", c.Destination.Name); err != nil {
		return err
	}
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.destination.detail", c.Destination.Detail); err != nil {
		return err
	}
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.destination.target", c.Destination.Target); err != nil {
		return err
	}
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.destination.target_name", c.Destination.TargetName); err != nil {
		return err
	}
	if err := h.reqHandler.FMV1VariableSetVariable(ctx, c.ActiveFlowID, "voipbin.call.destination.type", string(c.Destination.Type)); err != nil {
		return err
	}

	return nil
}
