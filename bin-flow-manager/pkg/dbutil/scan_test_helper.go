package dbutil

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// createMockRows creates a mock sql.Rows for testing
func createMockRows(t *testing.T, columns []string, values [][]interface{}) *sql.Rows {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	// Note: Don't defer db.Close() here - the returned rows need the connection to stay open

	rows := mock.NewRows(columns)
	for _, rowValues := range values {
		// Convert []interface{} to []driver.Value
		driverValues := make([]driver.Value, len(rowValues))
		for i, v := range rowValues {
			driverValues[i] = v
		}
		rows.AddRow(driverValues...)
	}

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("failed to create rows: %v", err)
	}

	return result
}
