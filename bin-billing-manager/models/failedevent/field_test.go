package failedevent

import (
	"testing"
)

func TestField_Constants(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected string
	}{
		{"id", FieldID, "id"},
		{"retry_count", FieldRetryCount, "retry_count"},
		{"next_retry_at", FieldNextRetryAt, "next_retry_at"},
		{"status", FieldStatus, "status"},
		{"tm_update", FieldTMUpdate, "tm_update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.field) != tt.expected {
				t.Errorf("Field = %s, expected %s", tt.field, tt.expected)
			}
		})
	}
}
