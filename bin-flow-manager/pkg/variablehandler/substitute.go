package variablehandler

import (
	"context"
	"fmt"
	"strings"

	"monorepo/bin-flow-manager/models/variable"
)

// SubstituteString substitutes the given data string with variables
func (h *variableHandler) SubstituteString(ctx context.Context, data string, v *variable.Variable) string {

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

// Substitute substitutes the given data with variables
func (h *variableHandler) SubstituteByte(ctx context.Context, data []byte, v *variable.Variable) []byte {

	tmp := string(data)
	res := h.SubstituteString(ctx, tmp, v)
	return []byte(res)
}
