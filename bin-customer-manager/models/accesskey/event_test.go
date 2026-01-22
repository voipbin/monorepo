package accesskey

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_accesskey_created", EventTypeAccesskeyCreated, "accesskey_created"},
		{"event_type_accesskey_updated", EventTypeAccesskeyUpdated, "accesskey_updated"},
		{"event_type_accesskey_deleted", EventTypeAccesskeyDeleted, "accesskey_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
