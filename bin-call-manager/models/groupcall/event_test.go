package groupcall

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_groupcall_created", EventTypeGroupcallCreated, "groupcall_created"},
		{"event_type_groupcall_progressing", EventTypeGroupcallProgressing, "groupcall_progressing"},
		{"event_type_groupcall_hangup", EventTypeGroupcallHangup, "groupcall_hangup"},
		{"event_type_groupcall_deleted", EventTypeGroupcallDeleted, "groupcall_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
