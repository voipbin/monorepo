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
			name:     "event_type_message_created",
			constant: EventTypeMessageCreated,
			expected: "conversation_message_created",
		},
		{
			name:     "event_type_message_updated",
			constant: EventTypeMessageUpdated,
			expected: "conversation_message_updated",
		},
		{
			name:     "event_type_message_deleted",
			constant: EventTypeMessageDeleted,
			expected: "conversation_message_deleted",
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
