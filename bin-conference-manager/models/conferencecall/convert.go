package conferencecall

import (
	"github.com/gofrs/uuid"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with conferencecall.Field keys,
// performing necessary type coercions for specific conferencecall fields.
func ConvertStringMapToFieldMap(src map[string]string) (map[Field]any, error) {
	res := make(map[Field]any, len(src))

	for key, val := range src {
		field := Field(key) // Convert the string key to the conferencecall.Field type

		switch field {
		case FieldDeleted:
			res[field] = val == "true"

		// Fields requiring UUID conversion
		case FieldID, FieldCustomerID, FieldActiveflowID, FieldConferenceID, FieldReferenceID:
			id := uuid.FromStringOrNil(val)
			res[field] = id

		case FieldReferenceType:
			res[field] = ReferenceType(val)

		case FieldStatus:
			res[field] = Status(val)

		// Fields that are expected to be strings in the model.
		case FieldTMCreate, FieldTMUpdate, FieldTMDelete:
			res[field] = val

		default:
			// For unknown fields, just pass through as string
			res[field] = val
		}
	}

	return res, nil
}
