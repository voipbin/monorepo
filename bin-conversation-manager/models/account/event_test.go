package account

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "event_type_account_created",
			constant: EventTypeAccountCreated,
			expected: "account_created",
		},
		{
			name:     "event_type_account_updated",
			constant: EventTypeAccountUpdated,
			expected: "account_updated",
		},
		{
			name:     "event_type_account_deleted",
			constant: EventTypeAccountDeleted,
			expected: "account_deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
