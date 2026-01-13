package conference

import (
	"github.com/gofrs/uuid"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with conference.Field keys,
// performing necessary type coercions for specific conference fields.
func ConvertStringMapToFieldMap(src map[string]string) (map[Field]any, error) {
	res := make(map[Field]any, len(src))

	for key, val := range src {
		field := Field(key) // Convert the string key to the conference.Field type

		switch field {
		case FieldDeleted:
			res[field] = val == "true"

		// Fields requiring UUID conversion
		case FieldID, FieldCustomerID, FieldConfbridgeID, FieldPreFlowID, FieldPostFlowID, FieldRecordingID, FieldTranscribeID:
			id := uuid.FromStringOrNil(val)
			res[field] = id

		case FieldType:
			res[field] = Type(val)

		case FieldStatus:
			res[field] = Status(val)

		// Fields that are expected to be strings in the model.
		case FieldName, FieldDetail, FieldTMEnd, FieldTMCreate, FieldTMUpdate, FieldTMDelete:
			res[field] = val

		default:
			// For unknown fields, just pass through as string
			res[field] = val
		}
	}

	return res, nil
}
