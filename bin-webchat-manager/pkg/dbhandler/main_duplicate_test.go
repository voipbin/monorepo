package dbhandler

import (
	"fmt"
	"testing"
)

// Test_IsErrDuplicate mirrors bin-ai-manager's equivalent. The helper exists so
// sessionhandler can tell "this id is taken" from any other insert failure; if
// it stopped matching, a caller-specified id collision would fall through to
// listenhandler's untyped 500 instead of the 409 the client needs.
func Test_IsErrDuplicate(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		expectRes bool
	}{
		{
			name:      "nil error",
			err:       nil,
			expectRes: false,
		},
		{
			name: "mysql duplicate entry",
			// The production driver's wording.
			err:       fmt.Errorf("could not execute query. SessionCreate. err: Error 1062 (23000): Duplicate entry 'x' for key 'PRIMARY'"),
			expectRes: true,
		},
		{
			name: "sqlite unique constraint",
			// The test suite's driver reports it differently.
			err:       fmt.Errorf("could not execute query. SessionCreate. err: UNIQUE constraint failed: webchat_sessions.id"),
			expectRes: true,
		},
		{
			name:      "unrelated failure",
			err:       fmt.Errorf("could not execute query. SessionCreate. err: connection refused"),
			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := IsErrDuplicate(tt.err); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
