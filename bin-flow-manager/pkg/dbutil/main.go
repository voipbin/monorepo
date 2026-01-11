package dbutil

import (
	"database/sql"
)

// GetDBFields returns ordered column names from struct tags
func GetDBFields(model interface{}) []string {
	panic("not implemented")
}

// PrepareValues converts struct fields to database values for INSERT/UPDATE
func PrepareValues(model interface{}) ([]interface{}, error) {
	panic("not implemented")
}

// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
func ScanRow(row *sql.Rows, dest interface{}) error {
	panic("not implemented")
}
