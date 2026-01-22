package messagechatroom

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_messagechatroom_created", EventTypeMessagechatroomCreated, "messagechatroom_created"},
		{"event_type_messagechatroom_updated", EventTypeMessagechatroomUpdated, "messagechatroom_updated"},
		{"event_type_messagechatroom_deleted", EventTypeMessagechatroomDeleted, "messagechatroom_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
