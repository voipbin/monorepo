package activeflowhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-flow-manager/models/variable"
)

// variableSubstitueAddress substitue the address with variables
func (h *activeflowHandler) variableSubstitueAddress(ctx context.Context, address *commonaddress.Address, v *variable.Variable) {
	address.Name = h.variableHandler.SubstituteString(ctx, address.Name, v)
	address.Detail = h.variableHandler.SubstituteString(ctx, address.Detail, v)
	address.Target = h.variableHandler.SubstituteString(ctx, address.Target, v)
	address.TargetName = h.variableHandler.SubstituteString(ctx, address.TargetName, v)
}
