package pod

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name string

		eventType     string
		expectedValue string
	}{
		{
			name: "pod_added_event_type",

			eventType:     EventTypePodAdded,
			expectedValue: "pod_added",
		},
		{
			name: "pod_updated_event_type",

			eventType:     EventTypePodUpdated,
			expectedValue: "pod_updated",
		},
		{
			name: "pod_deleted_event_type",

			eventType:     EventTypePodDeleted,
			expectedValue: "pod_deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.eventType != tt.expectedValue {
				t.Errorf("Wrong event type. expect: %s, got: %s", tt.expectedValue, tt.eventType)
			}
		})
	}
}

func TestEventTypeUniqueness(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "all_event_types_are_unique",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventTypes := map[string]bool{
				EventTypePodAdded:   true,
				EventTypePodUpdated: true,
				EventTypePodDeleted: true,
			}

			if len(eventTypes) != 3 {
				t.Errorf("Expected 3 unique event types, got: %d", len(eventTypes))
			}
		})
	}
}
