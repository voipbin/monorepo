package variablehandler

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
)

// Substitue substitues the given data with variables
func (h *variableHandler) Substitue(ctx context.Context, data string, v *variable.Variable) string {

	res := data

	targets := strings.Split(data, "${")
	for _, t := range targets {
		idx := strings.Index(t, "}")
		if idx < 0 {
			continue
		}

		target := t[:idx]
		variable := fmt.Sprintf("${%s}", target)
		value := v.Variables[target]
		res = strings.ReplaceAll(res, variable, value)
	}

	return res
}
