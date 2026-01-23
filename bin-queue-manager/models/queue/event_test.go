package queue

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_queue_created", EventTypeQueueCreated, "queue_created"},
		{"event_type_queue_updated", EventTypeQueueUpdated, "queue_updated"},
		{"event_type_queue_deleted", EventTypeQueueDeleted, "queue_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
