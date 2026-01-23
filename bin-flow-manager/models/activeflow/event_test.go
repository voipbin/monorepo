package activeflow

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_activeflow_created", EventTypeActiveflowCreated, "activeflow_created"},
		{"event_type_activeflow_updated", EventTypeActiveflowUpdated, "activeflow_updated"},
		{"event_type_activeflow_deleted", EventTypeActiveflowDeleted, "activeflow_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
