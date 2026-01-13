package file

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with file.Field keys,
// performing necessary type coercions for specific file fields.
func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	res := make(map[Field]any, len(src))

	for key, val := range src {
		field := Field(key) // Convert the string key to the file.Field type

		switch field {
		case FieldDeleted:
			parsed, ok := val.(bool)
			if !ok {
				return nil, fmt.Errorf("expected bool for %s", key)
			}
			res[field] = parsed

		// Fields requiring UUID conversion
		case FieldID, FieldCustomerID, FieldOwnerID, FieldAccountID, FieldReferenceID:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for uuid field %s, got %T", key, val)
			}
			id := uuid.FromStringOrNil(str)
			res[field] = id

		case FieldReferenceType:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for type field %s, got %T", key, val)
			}
			res[field] = ReferenceType(str)

		default:
			// For unknown fields, pass through as-is (strings, etc.)
			res[field] = val
		}
	}

	return res, nil
}
