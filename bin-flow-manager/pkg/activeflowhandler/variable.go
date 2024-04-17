package activeflowhandler

import (
	"context"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
)

// variableSubstitueAddress substitue the address with variables
func (h *activeflowHandler) variableSubstitueAddress(ctx context.Context, address *commonaddress.Address, v *variable.Variable) {
	address.Name = h.variableHandler.SubstituteString(ctx, address.Name, v)
	address.Detail = h.variableHandler.SubstituteString(ctx, address.Detail, v)
	address.Target = h.variableHandler.SubstituteString(ctx, address.Target, v)
	address.TargetName = h.variableHandler.SubstituteString(ctx, address.TargetName, v)
}
