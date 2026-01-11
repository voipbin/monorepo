package dbutil

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofrs/uuid"
)

// GetDBFields returns ordered column names from struct tags
func GetDBFields(model interface{}) []string {
	return getDBFieldsRecursive(reflect.ValueOf(model))
}

// getDBFieldsRecursive is the internal recursive function that works with reflect.Value
func getDBFieldsRecursive(val reflect.Value) []string {
	// Dereference pointer if needed
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Must be a struct at this point
	if val.Kind() != reflect.Struct {
		return []string{}
	}

	typ := val.Type()
	fields := []string{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Handle embedded structs recursively
		if field.Anonymous {
			embeddedVal := val.Field(i)
			embeddedFields := getDBFieldsRecursive(embeddedVal)
			fields = append(fields, embeddedFields...)
			continue
		}

		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: "column_name" or "column_name,conversion_type"
		parts := strings.Split(tag, ",")
		columnName := parts[0]

		fields = append(fields, columnName)
	}

	return fields
}

// PrepareValues converts struct fields to database values for INSERT/UPDATE
func PrepareValues(model interface{}) ([]interface{}, error) {
	return prepareValuesRecursive(reflect.ValueOf(model))
}

// prepareValuesRecursive is the internal recursive implementation
func prepareValuesRecursive(val reflect.Value) ([]interface{}, error) {
	// Handle pointer dereferencing
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Return empty slice for non-struct types
	if val.Kind() != reflect.Struct {
		return []interface{}{}, nil
	}

	typ := val.Type()
	values := []interface{}{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Handle embedded structs recursively
		if field.Anonymous {
			embeddedVal := val.Field(i)
			embeddedValues, err := prepareValuesRecursive(embeddedVal)
			if err != nil {
				return nil, err
			}
			values = append(values, embeddedValues...)
			continue
		}

		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: "column_name" or "column_name,conversion_type"
		parts := strings.Split(tag, ",")
		conversionType := ""
		if len(parts) > 1 {
			conversionType = parts[1]
		}

		fieldVal := val.Field(i)

		// Apply conversions based on type
		switch conversionType {
		case "uuid":
			// Convert uuid.UUID to []byte
			if fieldVal.Type() == reflect.TypeOf(uuid.UUID{}) {
				uuidVal := fieldVal.Interface().(uuid.UUID)
				values = append(values, uuidVal.Bytes())
			} else {
				return nil, fmt.Errorf("field %s: expected uuid.UUID type for uuid conversion", field.Name)
			}
		default:
			values = append(values, fieldVal.Interface())
		}
	}

	return values, nil
}

// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
func ScanRow(row *sql.Rows, dest interface{}) error {
	panic("not implemented")
}
