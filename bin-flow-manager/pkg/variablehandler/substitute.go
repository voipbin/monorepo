package variablehandler

import (
	"context"
	"fmt"
	"strings"

	"monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// Substitute substitutes the given data string with variables
func (h *variableHandler) Substitute(ctx context.Context, id uuid.UUID, data string) (string, error) {

	vars, err := h.Get(ctx, id)
	if err != nil {
		return "", errors.Wrapf(err, "could not get variable info. id: %s", id)
	}

	res := h.SubstituteString(ctx, data, vars)
	return res, nil
}

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

// Substitute substitutes the given data with variables
func (h *variableHandler) SubstituteOption(ctx context.Context, data map[string]any, vars *variable.Variable) {

	for k, v := range data {
		switch v := v.(type) {
		case string:
			data[k] = h.SubstituteString(ctx, v, vars)
		case []byte:
			data[k] = h.SubstituteByte(ctx, v, vars)
		default:
			continue
		}
	}
}
