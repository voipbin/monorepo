package outplan

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_outplan_created", EventTypeOutplanCreated, "outplan_created"},
		{"event_type_outplan_updated", EventTypeOutplanUpdated, "outplan_updated"},
		{"event_type_outplan_deleted", EventTypeOutplanDeleted, "outplan_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
