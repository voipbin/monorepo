package conversation

import (
	"encoding/json"
	"fmt"
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

func ConvertSringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	res := make(map[Field]any, len(src))

	for key, val := range src {
		field := Field(key)

		switch field {
		case FieldDeleted:
			parsed, ok := val.(bool)
			if !ok {
				return nil, fmt.Errorf("expected bool for %s", key)
			}
			res[field] = parsed

		case FieldID, FieldCustomerID, FieldOwnerID, FieldAccountID:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for uuid field %s", key)
			}
			id := uuid.FromStringOrNil(str)
			res[field] = id

		case FieldType:
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for type field %s", key)
			}
			res[field] = Type(str)

		case FieldSelf, FieldPeer:
			var addr commonaddress.Address
			switch v := val.(type) {
			case string:
				if err := json.Unmarshal([]byte(v), &addr); err != nil {
					return nil, fmt.Errorf("invalid address JSON string: %v", err)
				}
			case map[string]any:
				b, _ := json.Marshal(v)
				if err := json.Unmarshal(b, &addr); err != nil {
					return nil, fmt.Errorf("invalid address object: %v", err)
				}
			default:
				return nil, fmt.Errorf("unsupported address type for %s", key)
			}
			res[field] = addr

		default:
			res[field] = val
		}
	}

	return res, nil
}
