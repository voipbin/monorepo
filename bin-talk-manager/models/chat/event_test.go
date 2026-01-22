package chat

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_chat_created", EventTypeChatCreated, "chat_created"},
		{"event_type_chat_updated", EventTypeChatUpdated, "chat_updated"},
		{"event_type_chat_deleted", EventTypeChatDeleted, "chat_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
