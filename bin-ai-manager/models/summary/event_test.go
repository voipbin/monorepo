package summary

import "testing"

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "event_type_created",
			constant: EventTypeCreated,
			expected: "summary_created",
		},
		{
			name:     "event_type_updated",
			constant: EventTypeUpdated,
			expected: "summary_updated",
		},
		{
			name:     "event_type_deleted",
			constant: EventTypeDeleted,
			expected: "summary_deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
