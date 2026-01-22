package streaming

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
			name:     "event_type_streaming_finished",
			constant: EventTypeStreamingFinished,
			expected: "streaming_finished",
		},
		{
			name:     "event_type_streaming_created",
			constant: EventTypeStreamingCreated,
			expected: "streaming_created",
		},
		{
			name:     "event_type_streaming_deleted",
			constant: EventTypeStreamingDeleted,
			expected: "streaming_deleted",
		},
		{
			name:     "event_type_streaming_play_started",
			constant: EventTypeStreamingPlayStarted,
			expected: "streaming_play_started",
		},
		{
			name:     "event_type_streaming_play_finished",
			constant: EventTypeStreamingPlayFinished,
			expected: "streaming_play_finished",
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
