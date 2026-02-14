package tag

import "testing"

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		expected string
	}{
		{
			name:     "event_type_tag_created",
			event:    EventTypeTagCreated,
			expected: "tag_created",
		},
		{
			name:     "event_type_tag_updated",
			event:    EventTypeTagUpdated,
			expected: "tag_updated",
		},
		{
			name:     "event_type_tag_deleted",
			event:    EventTypeTagDeleted,
			expected: "tag_deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event != tt.expected {
				t.Errorf("Wrong event type. expect: %s, got: %s", tt.expected, tt.event)
			}
		})
	}
}
