package dbhandler

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func Test_isDuplicateKey(t *testing.T) {
	tests := []struct {
		name string
		err  error

		expect bool
	}{
		{name: "nil", err: nil, expect: false},
		{name: "mysql duplicate entry", err: &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}, expect: true},
		{name: "mysql wrapped duplicate entry", err: fmt.Errorf("wrapped: %w", &mysql.MySQLError{Number: 1062}), expect: true},
		{name: "mysql another error", err: &mysql.MySQLError{Number: 1146, Message: "Table doesn't exist"}, expect: false},
		{name: "mysql check constraint", err: &mysql.MySQLError{Number: 3819, Message: "Check constraint is violated"}, expect: false},
		{name: "sqlite unique constraint", err: errors.New("UNIQUE constraint failed: ai_ais.id"), expect: true},
		{name: "other error", err: errors.New("connection refused"), expect: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDuplicateKey(tt.err); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
			// the exported wrapper (the draft decision claim) answers the same.
			if got := IsDuplicateKey(tt.err); got != tt.expect {
				t.Errorf("Wrong match for IsDuplicateKey. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}
