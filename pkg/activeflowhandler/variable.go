package activeflowhandler

import (
	"context"
	"fmt"
	"strings"

	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// variableSubstitue substitue the given data with variables
func (h *activeflowHandler) variableSubstitue(ctx context.Context, data string, variables map[string]string) string {

	res := data

	targets := strings.Split(data, "${")
	for _, t := range targets {
		idx := strings.Index(t, "}")
		if idx < 0 {
			continue
		}

		target := t[:idx]
		variable := fmt.Sprintf("${%s}", target)
		value := variables[target]
		res = strings.ReplaceAll(res, variable, value)
	}

	return res
}

// variableSubstitueAddress substitue the address with variables
func (h *activeflowHandler) variableSubstitueAddress(ctx context.Context, address *cmaddress.Address, variables map[string]string) {
	address.Name = h.variableSubstitue(ctx, address.Name, variables)
	address.Detail = h.variableSubstitue(ctx, address.Detail, variables)
	address.Target = h.variableSubstitue(ctx, address.Target, variables)
	address.TargetName = h.variableSubstitue(ctx, address.TargetName, variables)
}
