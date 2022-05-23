package activeflowhandler

import (
	"context"
	"fmt"
	"strings"
)

// Substitue substitue the given or
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
