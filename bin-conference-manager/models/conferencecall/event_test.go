package conferencecall

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_conferencecall_joining", EventTypeConferencecallJoining, "conferencecall_joining"},
		{"event_type_conferencecall_joined", EventTypeConferencecallJoined, "conferencecall_joined"},
		{"event_type_conferencecall_leaving", EventTypeConferencecallLeaving, "conferencecall_leaving"},
		{"event_type_conferencecall_leaved", EventTypeConferencecallLeaved, "conferencecall_leaved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
