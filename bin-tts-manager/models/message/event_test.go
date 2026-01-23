package message

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
			name:     "event_type_initiated",
			constant: EventTypeInitiated,
			expected: "message_initiated",
		},
		{
			name:     "event_type_play_started",
			constant: EventTypePlayStarted,
			expected: "message_play_started",
		},
		{
			name:     "event_type_play_finished",
			constant: EventTypePlayFinished,
			expected: "message_play_finished",
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
