package activeflowhandler

import (
	"context"

	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
)

// variableSubstitueAddress substitue the address with variables
func (h *activeflowHandler) variableSubstitueAddress(ctx context.Context, address *cmaddress.Address, v *variable.Variable) {
	address.Name = h.variableHandler.Substitue(ctx, address.Name, v)
	address.Detail = h.variableHandler.Substitue(ctx, address.Detail, v)
	address.Target = h.variableHandler.Substitue(ctx, address.Target, v)
	address.TargetName = h.variableHandler.Substitue(ctx, address.TargetName, v)
}
