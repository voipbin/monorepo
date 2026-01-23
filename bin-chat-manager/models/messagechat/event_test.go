package messagechat

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_messagechat_created", EventTypeMessagechatCreated, "messagechat_created"},
		{"event_type_messagechat_updated", EventTypeMessagechatUpdated, "messagechat_updated"},
		{"event_type_messagechat_deleted", EventTypeMessagechatDeleted, "messagechat_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
