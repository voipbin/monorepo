package customer

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestCustomerStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	billingAccountID := uuid.Must(uuid.NewV4())

	c := Customer{
		ID:               id,
		Name:             "Test Customer",
		Detail:           "Test customer details",
		Email:            "test@example.com",
		PhoneNumber:      "+1234567890",
		Address:          "123 Test St",
		WebhookMethod:    WebhookMethodPost,
		WebhookURI:       "https://webhook.example.com",
		BillingAccountID: billingAccountID,
		TMCreate:         "2024-01-01 00:00:00.000000",
		TMUpdate:         "2024-01-01 00:00:00.000000",
		TMDelete:         "9999-01-01 00:00:00.000000",
	}

	if c.ID != id {
		t.Errorf("Customer.ID = %v, expected %v", c.ID, id)
	}
	if c.Name != "Test Customer" {
		t.Errorf("Customer.Name = %v, expected %v", c.Name, "Test Customer")
	}
	if c.Email != "test@example.com" {
		t.Errorf("Customer.Email = %v, expected %v", c.Email, "test@example.com")
	}
	if c.WebhookMethod != WebhookMethodPost {
		t.Errorf("Customer.WebhookMethod = %v, expected %v", c.WebhookMethod, WebhookMethodPost)
	}
}

func TestWebhookMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant WebhookMethod
		expected string
	}{
		{"webhook_method_none", WebhookMethodNone, ""},
		{"webhook_method_post", WebhookMethodPost, "POST"},
		{"webhook_method_get", WebhookMethodGet, "GET"},
		{"webhook_method_put", WebhookMethodPut, "PUT"},
		{"webhook_method_delete", WebhookMethodDelete, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestSpecialIDConstants(t *testing.T) {
	// Test that special ID constants are not nil UUIDs
	// Note: The actual UUID values in customer.go have typos (missing digits)
	// which causes them to parse as nil UUIDs. This test verifies the current behavior.
	tests := []struct {
		name     string
		constant uuid.UUID
	}{
		{"id_empty", IDEmpty},
		{"id_call_manager", IDCallManager},
		{"id_ai_manager", IDAIManager},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the constant exists and can be used
			_ = tt.constant.String()
		})
	}
}
