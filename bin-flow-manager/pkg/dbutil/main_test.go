package dbutil

import (
	"testing"

	"github.com/gofrs/uuid"
)

// Test model for validating dbutil functions
type testModel struct {
	ID     uuid.UUID `db:"id,uuid"`
	Name   string    `db:"name"`
	Count  int       `db:"count"`
	SkipMe bool      `db:"-"`
}

func TestGetDBFields_Basic(t *testing.T) {
	t.Skip("Not implemented yet")
}

func TestPrepareValues_Basic(t *testing.T) {
	t.Skip("Not implemented yet")
}

func TestScanRow_Basic(t *testing.T) {
	t.Skip("Not implemented yet")
}
