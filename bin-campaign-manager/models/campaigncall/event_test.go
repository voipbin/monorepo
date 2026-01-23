package campaigncall

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_campaigncall_created", EventTypeCampaigncallCreated, "campaigncall_created"},
		{"event_type_campaigncall_updated", EventTypeCampaigncallUpdated, "campaigncall_updated"},
		{"event_type_campaigncall_deleted", EventTypeCampaigncallDeleted, "campaigncall_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
