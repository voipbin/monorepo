package variablehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-flow-manager/models/activeflow"
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

	res := h.substituteString(ctx, data, vars)
	return res, nil
}

// substituteString substitutes the given data string with variables
func (h *variableHandler) substituteString(ctx context.Context, data string, v *variable.Variable) string {
	return regexVariable.ReplaceAllStringFunc(data, func(match string) string {
		submatches := regexVariable.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return ""
		}
		variableName := submatches[1] // Second submatch is the variable name

		// Try to get the value from the static variable first
		if value, ok := h.substituteParseStatic(ctx, variableName, v); ok {
			return value
		}

		// Try to get the value from dynamic resource
		if value, ok := h.substituteParseDynamic(ctx, variableName, v); ok {
			return value
		}

		return ""
	})
}

func (h *variableHandler) substituteParseStatic(ctx context.Context, variableName string, v *variable.Variable) (string, bool) {
	value, ok := v.Variables[variableName]
	if ok {
		return value, true
	}

	return "", false
}

// Substitute substitutes the given data with variables
func (h *variableHandler) substituteByte(ctx context.Context, data []byte, v *variable.Variable) []byte {
	tmp := string(data)
	res := h.substituteString(ctx, tmp, v)
	return []byte(res)
}

// SubstituteOption substitutes the given data with variables, supporting nested maps, slices, and pointers
func (h *variableHandler) SubstituteOption(ctx context.Context, data map[string]any, vars *variable.Variable) {
	for k, v := range data {
		switch v := v.(type) {
		case string:
			data[k] = h.substituteString(ctx, v, vars)
		case []byte:
			data[k] = h.substituteByte(ctx, v, vars)
		case map[string]any:
			h.SubstituteOption(ctx, v, vars)
		case []any:
			for i, elem := range v {
				data[k].([]any)[i] = h.resolveValue(ctx, elem, vars)
			}
		case []string:
			for i, elem := range v {
				data[k].([]string)[i] = h.substituteString(ctx, elem, vars)
			}
		case []map[string]any:
			for i, m := range v {
				h.SubstituteOption(ctx, m, vars)
				data[k].([]map[string]any)[i] = m
			}
		case *string:
			data[k] = h.resolveValue(ctx, *v, vars)
		case *map[string]any:
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
		return h.substituteString(ctx, v, vars)
	case []byte:
		return h.substituteByte(ctx, v, vars)
	case map[string]any:
		h.SubstituteOption(ctx, v, vars)
		return v
	case []any:
		for i, elem := range v {
			v[i] = h.resolveValue(ctx, elem, vars)
		}
		return v
	case []string:
		for i, elem := range v {
			v[i] = h.substituteString(ctx, elem, vars)
		}
		return v
	case *string:
		return h.resolveValue(ctx, *v, vars)
	case *map[string]any:
		return h.resolveValue(ctx, *v, vars)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (h *variableHandler) substituteParseDynamic(ctx context.Context, variableName string, v *variable.Variable) (string, bool) {

	switch variableName {
	case constVariableReferenceData:
		af, err := h.requestHandler.FlowV1ActiveflowGet(ctx, v.ID)
		if err != nil {
			return "", false
		}

		switch af.ReferenceType {
		case activeflow.ReferenceTypeCall:
			c, err := h.requestHandler.CallV1CallGet(ctx, af.ReferenceID)
			if err != nil {
				return "", false
			}

			cc := c.ConvertWebhookMessage()
			tmp, err := json.Marshal(cc)
			if err != nil {
				return "", false
			}

			res := string(tmp)
			return res, true

		case activeflow.ReferenceTypeConversation:
			c, err := h.requestHandler.ConversationV1ConversationGet(ctx, af.ReferenceID)
			if err != nil {
				return "", false
			}

			cc := c.ConvertWebhookMessage()
			tmp, err := json.Marshal(cc)
			if err != nil {
				return "", false
			}

			res := string(tmp)
			return res, true

		default:
			return "", false
		}

	default:
		return "", false
	}
}
