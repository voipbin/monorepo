package databasehandler

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

const (
	DefaultTimeStamp = "9999-01-01 00:00:000"
)

func PrepareUpdateFields[K ~string](fields map[K]any) map[string]any {
	res := make(map[string]any, len(fields))
	for k, v := range fields {
		key := string(k)

		switch val := v.(type) {
		case uuid.UUID:
			res[key] = val.Bytes()

		case json.Marshaler:
			b, err := val.MarshalJSON()
			if err == nil {
				res[key] = b
			} else {
				res[key] = nil
			}

		default:
			rv := reflect.ValueOf(v)
			rt := rv.Type()
			if rt.Kind() == reflect.Map || rt.Kind() == reflect.Slice || rt.Kind() == reflect.Struct {
				b, err := json.Marshal(v)
				if err == nil {
					res[key] = b
				} else {
					res[key] = nil
				}
			} else {
				res[key] = v
			}
		}
	}

	return res
}

func ApplyFields[K ~string](sb squirrel.SelectBuilder, fields map[K]any) (squirrel.SelectBuilder, error) {
	for k, v := range fields {
		key := string(k)

		switch val := v.(type) {

		case uuid.UUID:
			sb = sb.Where(squirrel.Eq{key: val.Bytes()})

		case string:
			sb = sb.Where(squirrel.Eq{key: val})

		case int, int64, uint, uint64, float32, float64:
			sb = sb.Where(squirrel.Eq{key: val})

		case bool:
			if key == "deleted" && !val {
				sb = sb.Where(squirrel.GtOrEq{"tm_delete": DefaultTimeStamp})
			} else {
				sb = sb.Where(squirrel.Eq{key: val})
			}

		default:
			return sb, fmt.Errorf("unsupported filter type for %s: %T", key, v)
		}
	}

	return sb, nil
}
