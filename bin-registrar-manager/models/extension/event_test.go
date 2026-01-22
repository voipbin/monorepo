package extension

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_extension_created", EventTypeExtensionCreated, "extension_created"},
		{"event_type_extension_updated", EventTypeExtensionUpdated, "extension_updated"},
		{"event_type_extension_deleted", EventTypeExtensionDeleted, "extension_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
