package queuecall

import (
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-queue-manager/models/queue"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with queuecall.Field keys,
// performing necessary type coercions for specific queuecall fields.
func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	res := make(map[Field]any, len(src))

	for key, val := range src {
		field := Field(key) // Convert the string key to the queuecall.Field type

		switch field {
		case FieldDeleted:
			// Handle both bool and string types for deleted filter
			switch v := val.(type) {
			case bool:
				res[field] = v
			case string:
				// Convert string "true"/"false" to boolean
				res[field] = v == "true"
			default:
				return nil, fmt.Errorf("expected bool or string for %s, got %T", key, val)
			}


		// Fields requiring UUID conversion
		case FieldID, FieldCustomerID, FieldQueueID, FieldReferenceID, FieldReferenceActiveflowID, FieldForwardActionID, FieldConfbridgeID, FieldServiceAgentID:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for uuid field %s, got %T", key, val)
			}
			id := uuid.FromStringOrNil(str)
			res[field] = id

		case FieldReferenceType:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for reference_type field %s, got %T", key, val)
			}
			res[field] = ReferenceType(str)

		case FieldRoutingMethod:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for routing_method field %s, got %T", key, val)
			}
			res[field] = queue.RoutingMethod(str)

		case FieldStatus:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for status field %s, got %T", key, val)
			}
			res[field] = Status(str)

		// Integer fields
		case FieldTimeoutWait, FieldTimeoutService, FieldDurationWaiting, FieldDurationService:
			// JSON numbers come as float64
			num, ok := val.(float64)
			if !ok {
				return nil, fmt.Errorf("expected number for field %s, got %T", key, val)
			}
			res[field] = int(num)

		// String fields
		case FieldTMCreate, FieldTMService, FieldTMUpdate, FieldTMEnd, FieldTMDelete:
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
