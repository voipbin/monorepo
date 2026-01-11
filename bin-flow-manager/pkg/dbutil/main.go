package dbutil

import (
	"database/sql"
	"reflect"
	"strings"
)

// GetDBFields returns ordered column names from struct tags
func GetDBFields(model interface{}) []string {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	fields := []string{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
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
	panic("not implemented")
}

// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
func ScanRow(row *sql.Rows, dest interface{}) error {
	panic("not implemented")
}
