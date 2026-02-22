package speaking

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
			name:     "event_type_speaking_started",
			constant: EventTypeSpeakingStarted,
			expected: "speaking_started",
		},
		{
			name:     "event_type_speaking_stopped",
			constant: EventTypeSpeakingStopped,
			expected: "speaking_stopped",
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
