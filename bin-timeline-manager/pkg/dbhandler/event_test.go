package dbhandler

import (
	"testing"
)

func TestBuildEventConditions(t *testing.T) {
	tests := []struct {
		name     string
		events   []string
		expected string
	}{
		{
			name:     "exact match",
			events:   []string{"activeflow_created"},
			expected: "event_type = 'activeflow_created'",
		},
		{
			name:     "wildcard prefix",
			events:   []string{"activeflow_*"},
			expected: "event_type LIKE 'activeflow_%'",
		},
		{
			name:     "multiple patterns",
			events:   []string{"activeflow_created", "flow_*"},
			expected: "event_type = 'activeflow_created' OR event_type LIKE 'flow_%'",
		},
		{
			name:     "wildcard all",
			events:   []string{"*"},
			expected: "",
		},
		{
			name:     "empty",
			events:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEventConditions(tt.events)
			if result != tt.expected {
				t.Errorf("buildEventConditions(%v) = %q, want %q", tt.events, result, tt.expected)
			}
		})
	}
}
