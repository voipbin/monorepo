package queuecall

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"queuecall_created", EventTypeQueuecallCreated, "queuecall_created"},
		{"queuecall_waiting", EventTypeQueuecallWaiting, "queuecall_waiting"},
		{"queuecall_connecting", EventTypeQueuecallConnecting, "queuecall_connecting"},
		{"queuecall_serviced", EventTypeQueuecallServiced, "queuecall_serviced"},
		{"queuecall_done", EventTypeQueuecallDone, "queuecall_done"},
		{"queuecall_abandoned", EventTypeQueuecallAbandoned, "queuecall_abandoned"},
		{"queuecall_deleted", EventTypeQueuecallDeleted, "queuecall_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
