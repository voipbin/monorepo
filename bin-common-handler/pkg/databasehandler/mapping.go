package databasehandler

import (
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
