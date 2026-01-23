package trunk

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_trunk_created", EventTypeTrunkCreated, "trunk_created"},
		{"event_type_trunk_updated", EventTypeTrunkUpdated, "trunk_updated"},
		{"event_type_trunk_deleted", EventTypeTrunkDeleted, "trunk_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
