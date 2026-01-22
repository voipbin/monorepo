package flow

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_flow_created", EventTypeFlowCreated, "flow_created"},
		{"event_type_flow_updated", EventTypeFlowUpdated, "flow_updated"},
		{"event_type_flow_deleted", EventTypeFlowDeleted, "flow_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
