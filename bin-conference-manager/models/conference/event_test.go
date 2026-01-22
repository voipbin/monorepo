package conference

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_conference_created", EventTypeConferenceCreated, "conference_created"},
		{"event_type_conference_deleted", EventTypeConferenceDeleted, "conference_deleted"},
		{"event_type_conference_updated", EventTypeConferenceUpdated, "conference_updated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
