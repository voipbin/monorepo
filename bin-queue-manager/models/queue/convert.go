package queue

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with queue.Field keys,
// performing necessary type coercions for specific queue fields.
func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	res := make(map[Field]any, len(src))

	for key, val := range src {
		field := Field(key) // Convert the string key to the queue.Field type

		switch field {
		case FieldDeleted:
			parsed, ok := val.(bool)
			if !ok {
				return nil, fmt.Errorf("expected bool for %s", key)
			}
			res[field] = parsed

		// Fields requiring UUID conversion
		case FieldID, FieldCustomerID, FieldWaitFlowID:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for uuid field %s, got %T", key, val)
			}
			id := uuid.FromStringOrNil(str)
			res[field] = id

		case FieldRoutingMethod:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for routing_method field %s, got %T", key, val)
			}
			res[field] = RoutingMethod(str)

		case FieldExecute:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for execute field %s, got %T", key, val)
			}
			res[field] = Execute(str)

		// Integer fields
		case FieldWaitTimeout, FieldServiceTimeout, FieldTotalIncomingCount, FieldTotalServicedCount, FieldTotalAbandonedCount:
			// JSON numbers come as float64
			num, ok := val.(float64)
			if !ok {
				return nil, fmt.Errorf("expected number for field %s, got %T", key, val)
			}
			res[field] = int(num)

		// String fields
		case FieldName, FieldDetail, FieldTMCreate, FieldTMUpdate, FieldTMDelete:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for field %s, got %T", key, val)
			}
			res[field] = str

		// Default case: if the field key is not one of the recognized Field constants.
		default:
			return nil, fmt.Errorf("unknown or unhandled field: %s", key)
		}
	}

	return res, nil
}
