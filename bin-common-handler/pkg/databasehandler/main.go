package databasehandler

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultTimeStamp is a placeholder for a default timestamp value used in specific queries.
	// Uses ISO 8601 format with microsecond precision.
	DefaultTimeStamp = "9999-01-01T00:00:00.000000Z"
)

func Connect(dsn string) (*sql.DB, error) {
	res, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "database open error")
	}

	if err := res.Ping(); err != nil {
		return nil, errors.Wrap(err, "database ping error")
	}

	return res, nil
}

func Close(db *sql.DB) {
	if db == nil {
		return
	}

	if errClose := db.Close(); errClose != nil {
		logrus.Errorf("Could not close the database connection err: %v", errClose)
		return
	}
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
