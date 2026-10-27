package dbhandler

import (
	"context"
	"strings"
	"testing"

	"monorepo/bin-direct-manager/models/direct"
)

// Test_DirectUpdate_fieldNotEncodable checks that an update whose field value cannot be
// prepared for the database is rejected before any statement is sent.
func Test_DirectUpdate_fieldNotEncodable(t *testing.T) {
	h, mock := newCtxTestHandler(t)

	// a slice is stored as JSON, and a channel inside it cannot be encoded
	fields := map[direct.Field]any{
		direct.FieldHash: []any{make(chan int)},
	}

	err := h.DirectUpdate(context.Background(), testRowID, fields)
	if err == nil || !strings.Contains(err.Error(), "could not prepare fields. DirectUpdate") {
		t.Fatalf("DirectUpdate() error = %v, want the field preparation context", err)
	}

	// no statement was expected, so any statement sent would have been reported by the mock
	if errExpect := mock.ExpectationsWereMet(); errExpect != nil {
		t.Errorf("unexpected database activity: %v", errExpect)
	}
}
