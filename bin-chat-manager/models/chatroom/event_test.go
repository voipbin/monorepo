package chatroom

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_chatroom_created", EventTypeChatroomCreated, "chatroom_created"},
		{"event_type_chatroom_updated", EventTypeChatroomUpdated, "chatroom_updated"},
		{"event_type_chatroom_deleted", EventTypeChatroomDeleted, "chatroom_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
