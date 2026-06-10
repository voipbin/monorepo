package variable

import (
	"encoding/json"
	"strconv"

	"github.com/gofrs/uuid"
)

// Variable struct
type Variable struct {
	ID        uuid.UUID         `json:"id"` // same with the activeflow id.
	Variables map[string]string `json:"variables"`
}

// ToStringMap converts a map[string]any into the map[string]string shape variables use.
//
// Scalar values (string, bool, float64, json.Number) are stringified; non-scalar values
// (object, array, null) are skipped silently. Numbers decoded via a json.Decoder with
// UseNumber arrive as json.Number and are stringified without float precision loss; a plain
// json.Unmarshal yields float64, which is handled as a fallback. Returns nil for an empty or
// nil input (and for an input whose values are all non-scalar) so a downstream JSON marshal
// omits an empty variables field.
//
// This performs only type coercion. Reserved-key dropping and size caps are enforced
// elsewhere (flow-manager sanitizeInitialVariables), not here.
func ToStringMap(in map[string]any) map[string]string {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]string, len(in))
	for k, v := range in {
		switch val := v.(type) {
		case string:
			out[k] = val
		case bool:
			out[k] = strconv.FormatBool(val)
		case json.Number:
			out[k] = val.String()
		case float64:
			// Fallback when the source used a plain json.Unmarshal (numbers become float64).
			// Render integers without a trailing ".0".
			out[k] = strconv.FormatFloat(val, 'f', -1, 64)
		default:
			// object/array/null: skip silently.
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
