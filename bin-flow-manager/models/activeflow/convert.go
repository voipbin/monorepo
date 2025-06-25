package activeflow

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with account.Field keys,
// performing necessary type coercions for specific account fields.
func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	res := make(map[Field]any, len(src))

	for key, val := range src {
		field := Field(key) // Convert the string key to the account.Field type

		switch field {
		case FieldDeleted:
			parsed, ok := val.(bool)
			if !ok {
				return nil, fmt.Errorf("expected bool for %s", key)
			}
			res[field] = parsed

		// Fields requiring UUID conversion
		case FieldID, FieldCustomerID, FieldFlowID, FieldReferenceID, FieldReferenceActiveflowID, FieldCurrentStackID, FieldForwardStackID, FieldForwardActionID:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for uuid field %s, got %T", key, val)
			}
			id := uuid.FromStringOrNil(str)
			res[field] = id

		case FieldStatus:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for type field %s, got %T", key, val)
			}
			res[field] = Status(str)

		case FieldReferenceType:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for type field %s, got %T", key, val)
			}
			res[field] = ReferenceType(str)

		case FieldExecuteCount:
			num, ok := val.(float64) // Assuming the value is a number (e.g., from JSON)
			if !ok {
				return nil, fmt.Errorf("expected float64 for field %s, got %T", key, val)
			}
			res[field] = uint64(num) // Convert to uint64

		// Fields that are expected to be strings in the model.
		// These are handled explicitly to ensure type correctness,
		// rather than relying on a generic default passthrough, which can be risky.
		case FieldTMCreate, FieldTMUpdate, FieldTMDelete:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for field %s, got %T", key, val)
			}
			res[field] = str

		default:
			return nil, fmt.Errorf("unknown or unhandled field: %s", key)
		}
	}

	return res, nil
}
