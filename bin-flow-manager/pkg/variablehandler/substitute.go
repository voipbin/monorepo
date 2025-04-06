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

// SubstituteOption substitutes the given data with variables, supporting nested maps, slices, and pointers
func (h *variableHandler) SubstituteOption(ctx context.Context, data map[string]any, vars *variable.Variable) {
	for k, v := range data {
		switch v := v.(type) {
		case string:
			// Replace variable in string
			data[k] = h.SubstituteString(ctx, v, vars)
		case []byte:
			// Replace variable in byte array
			data[k] = h.SubstituteByte(ctx, v, vars)
		case map[string]any:
			// Recursively handle nested maps
			h.SubstituteOption(ctx, v, vars)
		case []any:
			// Handle slices/arrays, iterate over elements
			for i, elem := range v {
				// Recursively substitute elements in the slice
				data[k].([]any)[i] = h.resolveValue(ctx, elem, vars)
			}
		case []string:
			// Handle slices of strings, iterate over each string element
			for i, elem := range v {
				data[k].([]string)[i] = h.SubstituteString(ctx, elem, vars)
			}
		case []map[string]any:
			// Handle slices of maps, iterate over each map
			for i, m := range v {
				h.SubstituteOption(ctx, m, vars)  // Recursively substitute in each map
				data[k].([]map[string]any)[i] = m // Update the slice element with the substituted map
			}
		case *string:
			// Handle pointers to strings
			data[k] = h.resolveValue(ctx, *v, vars)
		case *map[string]any:
			// Handle pointers to maps
			data[k] = h.resolveValue(ctx, *v, vars)
		default:
			// For unsupported types, print a message and continue
			fmt.Printf("unsupported type %T for key: %s\n", v, k)
			continue
		}
	}
}

// resolveValue resolves any type to a string, including nested maps and slices
func (h *variableHandler) resolveValue(ctx context.Context, value any, vars *variable.Variable) any {
	switch v := value.(type) {
	case string:
		// Direct string, substitute variables
		return h.SubstituteString(ctx, v, vars)
	case []byte:
		// Byte array, substitute variables
		return h.SubstituteByte(ctx, v, vars)
	case map[string]any:
		// Recursively handle nested maps
		h.SubstituteOption(ctx, v, vars)
		return v
	case []any:
		// Handle slice/array
		for i, elem := range v {
			v[i] = h.resolveValue(ctx, elem, vars)
		}
		return v
	case []string:
		// Handle slice of strings
		for i, elem := range v {
			v[i] = h.SubstituteString(ctx, elem, vars)
		}
		return v
	case *string:
		// Pointer to string
		return h.resolveValue(ctx, *v, vars)
	case *map[string]any:
		// Pointer to map
		return h.resolveValue(ctx, *v, vars)
	default:
		// For unsupported types, return as string representation
		return fmt.Sprintf("%v", v)
	}
}
