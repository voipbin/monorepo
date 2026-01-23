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
		{"event_type_message_created", EventTypeMessageCreated, "message_created"},
		{"event_type_message_updated", EventTypeMessageUpdated, "message_updated"},
		{"event_type_message_deleted", EventTypeMessageDeleted, "message_ringing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
