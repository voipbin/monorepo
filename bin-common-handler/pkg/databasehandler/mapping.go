package databasehandler

import (
	"fmt"
	"reflect"
	"strings"
)

// GetDBFields returns ordered column names from struct tags
// Reads db:"column_name[,conversion_type]" tags from struct fields
// Skips fields tagged with db:"-"
// Recursively processes embedded structs
func GetDBFields(model interface{}) []string {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		// Check if pointer is nil
		if val.IsNil() {
			return []string{}
		}
		val = val.Elem()
	}

	typ := val.Type()
	return getDBFieldsRecursive(val, typ)
}

// getDBFieldsRecursive recursively extracts column names from struct fields
func getDBFieldsRecursive(val reflect.Value, typ reflect.Type) []string {
	var fields []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Handle embedded/anonymous structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			embeddedFields := getDBFieldsRecursive(fieldVal, field.Type)
			fields = append(fields, embeddedFields...)
			continue
		}

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get db tag
		tag := field.Tag.Get("db")
		if tag == "" {
			continue
		}

		// Skip fields with db:"-"
		if tag == "-" {
			continue
		}

		// Extract column name (before comma if conversion type specified)
		columnName := tag
		if idx := strings.Index(tag, ","); idx != -1 {
			columnName = tag[:idx]
		}

		fields = append(fields, columnName)
	}

	return fields
}

// PrepareFields converts struct or map to database-ready values
// - Struct input: reads db tags, skips db:"-", converts UUID/JSON based on tags
// - Map input: auto-detects types, converts UUID/JSON without tag filtering
// Returns map[string]any suitable for squirrel.Insert().SetMap() or Update().SetMap()
func PrepareFields(data any) (map[string]any, error) {
	val := reflect.ValueOf(data)

	// Dereference pointer if needed
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		// Tag-aware path for INSERT with structs
		return prepareFieldsFromStruct(val)

	case reflect.Map:
		// Tag-agnostic path for UPDATE with maps
		return prepareFieldsFromMap(data)

	default:
		return nil, fmt.Errorf("PrepareFields: expected struct or map, got %T", data)
	}
}
