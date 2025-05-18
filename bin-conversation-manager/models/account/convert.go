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
			// Using FromStringOrNil similar to the conversation example.
			// This means an invalid UUID string (that FromStringOrNil can't parse) becomes uuid.Nil.
			// If str is "" or "00000000-0000-0000-0000-000000000000", id will be uuid.Nil.
			id := uuid.FromStringOrNil(str)
			res[field] = id

		// Field requiring conversion to account.Type
		case FieldType:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for type field %s, got %T", key, val)
			}
			res[field] = Type(str) // Cast to account.Type

		// Fields that are expected to be strings in the model.
		// These are handled explicitly to ensure type correctness,
		// rather than relying on a generic default passthrough, which can be risky.
		case FieldName, FieldDetail, FieldSecret, FieldToken,
			FieldTMCreate, FieldTMUpdate, FieldTMDelete:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for field %s, got %T", key, val)
			}
			res[field] = str

		// Default case: if the field key is not one of the recognized account.Field constants.
		default:
			return nil, fmt.Errorf("unknown or unhandled field for Account: %s", key)
		}
	}

	return res, nil
}
