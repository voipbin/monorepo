package billing

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_billing_created", EventTypeBillingCreated, "billing_created"},
		{"event_type_billing_updated", EventTypeBillingUpdated, "billing_updated"},
		{"event_type_billing_deleted", EventTypeBillingDeleted, "billing_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
