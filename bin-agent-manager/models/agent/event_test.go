package agent

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
			name:     "event_type_agent_created",
			constant: EventTypeAgentCreated,
			expected: "agent_created",
		},
		{
			name:     "event_type_agent_updated",
			constant: EventTypeAgentUpdated,
			expected: "agent_updated",
		},
		{
			name:     "event_type_agent_deleted",
			constant: EventTypeAgentDeleted,
			expected: "agent_deleted",
		},
		{
			name:     "event_type_agent_status_updated",
			constant: EventTypeAgentStatusUpdated,
			expected: "agent_status_updated",
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
