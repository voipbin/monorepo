package account

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
		case FieldID, FieldCustomerID:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for uuid field %s, got %T", key, val)
			}
			id := uuid.FromStringOrNil(str)
			res[field] = id

		default:
			// For unknown fields, pass through as-is (strings, etc.)
			res[field] = val
		}
	}

	return res, nil
}
