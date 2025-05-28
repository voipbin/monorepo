package databasehandler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

const (
	// DefaultTimeStamp is a placeholder for a default timestamp value used in specific queries.
	DefaultTimeStamp = "9999-01-01 00:00:00.000000"
)

// PrepareUpdateFields processes a map of fields intended for an update operation (e.g., in a database).
// It converts specific types to a database-friendly format.
// - uuid.UUID values are converted to their byte representation.
// - Values implementing json.Marshaler are marshaled to JSON bytes.
// - Maps, slices, and structs are marshaled to JSON bytes if they don't implement json.Marshaler.
// - Other types are returned as is.
// If marshaling fails for a json.Marshaler or for map/slice/struct types, the value for that key
// in the result map is set to nil.
//
// The keys of the input map (type K, constrained to ~string) are converted to plain strings
// for the keys of the output map.
//
// Parameters:
//   - fields: A map where keys are of type K (a string or string-based custom type)
//     and values are of type any. These represent the fields to be updated.
//
// Returns:
//
//	A new map[string]any where values are processed according to the rules above.
//	This map is suitable for use with database update operations that expect
//	primitive or byte-slice values for complex types.
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

// ApplyFields dynamically adds WHERE clauses to a squirrel.SelectBuilder based on the provided fields map.
// It handles various data types for filter values:
// - uuid.UUID: Converted to bytes for equality comparison.
// - string, int, int64, uint, uint64, float32, float64: Used directly for equality comparison.
// - bool:
//   - If the key is "deleted" and value is false, it applies a special condition: "tm_delete" >= DefaultTimeStamp.
//   - Otherwise, used for equality comparison.
//   - Other string-based, integer-based, or float-based custom types (detected via reflection):
//     Converted to their base type for equality comparison.
//
// Parameters:
//   - sb: The squirrel.SelectBuilder to which WHERE clauses will be added.
//   - fields: A map where keys are of type K (a string or string-based custom type)
//     representing the database column names, and values are of type any
//     representing the filter values.
//
// Returns:
//
//	The modified squirrel.SelectBuilder with added WHERE clauses.
//	An error if an unsupported filter type is encountered.
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
			rv := reflect.ValueOf(v)

			switch rv.Kind() {
			case reflect.String:
				sb = sb.Where(squirrel.Eq{key: rv.String()})

			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				sb = sb.Where(squirrel.Eq{key: rv.Int()})

			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				sb = sb.Where(squirrel.Eq{key: rv.Uint()})

			case reflect.Float32, reflect.Float64:
				sb = sb.Where(squirrel.Eq{key: rv.Float()})

			default:
				return sb, fmt.Errorf("unsupported filter type for %s: %T (Kind: %s)", key, v, rv.Kind())
			}
		}
	}

	return sb, nil
}

// GetQuerySelectField takes a slice of any type T that has an underlying type of string
// (e.g., string, or custom types like `type MyField string`).
// It converts each non-empty element of the slice to its string representation
// and joins them into a single comma-separated string.
// This is typically used to generate the field list for a SQL SELECT statement.
//
// Parameters:
//   - fields: A slice of type []T, where T is a string or a string-based custom type.
//     Elements that convert to an empty string are ignored.
//
// Returns:
//
//	A string containing the non-empty field names, separated by ", ".
//	Returns an empty string if the input slice is nil or empty, or if all
//	elements convert to empty strings.
func GetQuerySelectField[T ~string](fields []T) string {
	if len(fields) == 0 {
		return ""
	}

	stringFields := make([]string, 0, len(fields))
	for _, item := range fields {
		strValue := string(item)

		if strValue != "" {
			stringFields = append(stringFields, strValue)
		}
	}

	return strings.Join(stringFields, ", ")
}
