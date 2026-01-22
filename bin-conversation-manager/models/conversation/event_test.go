package conversation

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
			name:     "event_type_conversation_created",
			constant: EventTypeConversationCreated,
			expected: "conversation_created",
		},
		{
			name:     "event_type_conversation_updated",
			constant: EventTypeConversationUpdated,
			expected: "conversation_updated",
		},
		{
			name:     "event_type_conversation_deleted",
			constant: EventTypeConversationDeleted,
			expected: "conversation_deleted",
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
