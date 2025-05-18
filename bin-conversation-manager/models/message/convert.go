package message

import (
	"encoding/json"
	"fmt"
	"monorepo/bin-conversation-manager/models/media"

	"github.com/gofrs/uuid"
)

func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	res := make(map[Field]any, len(src))

	for key, val := range src {
		field := Field(key)

		switch field {
		case FieldID, FieldCustomerID, FieldConversationID, FieldReferenceID:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for uuid field %s, got %T", key, val)
			}
			id := uuid.FromStringOrNil(str)
			res[field] = id

		case FieldDirection:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for direction field %s, got %T", key, val)
			}
			res[field] = Direction(str)
		case FieldStatus:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for status field %s, got %T", key, val)
			}
			res[field] = Status(str)
		case FieldReferenceType:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for reference_type field %s, got %T", key, val)
			}
			res[field] = ReferenceType(str)

		case FieldMedias:
			var medias []media.Media
			switch v := val.(type) {
			case string:
				if err := json.Unmarshal([]byte(v), &medias); err != nil {
					return nil, fmt.Errorf("invalid medias JSON string for %s: %v", key, err)
				}
			case []any:
				b, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("could not marshal medias array for %s: %v", key, err)
				}
				if err := json.Unmarshal(b, &medias); err != nil {
					return nil, fmt.Errorf("could not unmarshal medias array for %s: %v", key, err)
				}
			case nil:
				medias = []media.Media{}
			default:
				return nil, fmt.Errorf("unsupported type for medias field %s: got %T", key, val)
			}
			res[field] = medias

		case FieldTransactionID, FieldText,
			FieldTMCreate, FieldTMUpdate, FieldTMDelete:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for field %s, got %T", key, val)
			}
			res[field] = str

		case FieldDeleted:
			parsed, ok := val.(bool)
			if !ok {
				return nil, fmt.Errorf("expected bool for %s", key)
			}
			res[field] = parsed

		default:
			return nil, fmt.Errorf("unknown or unhandled field for Message: %s", key)
		}
	}

	return res, nil
}
