package extensiondirect

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_created", EventTypeExtensionDirectCreated, "extension_direct_created"},
		{"event_type_updated", EventTypeExtensionDirectUpdated, "extension_direct_updated"},
		{"event_type_deleted", EventTypeExtensionDirectDeleted, "extension_direct_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong event type constant. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
