package customer

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_customer_created", EventTypeCustomerCreated, "customer_created"},
		{"event_type_customer_updated", EventTypeCustomerUpdated, "customer_updated"},
		{"event_type_customer_deleted", EventTypeCustomerDeleted, "customer_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
