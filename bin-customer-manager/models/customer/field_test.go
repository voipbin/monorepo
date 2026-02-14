package customer

import (
	"testing"
)

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{"field_id", FieldID, "id"},
		{"field_name", FieldName, "name"},
		{"field_detail", FieldDetail, "detail"},
		{"field_email", FieldEmail, "email"},
		{"field_phone_number", FieldPhoneNumber, "phone_number"},
		{"field_address", FieldAddress, "address"},
		{"field_webhook_method", FieldWebhookMethod, "webhook_method"},
		{"field_webhook_uri", FieldWebhookURI, "webhook_uri"},
		{"field_billing_account_id", FieldBillingAccountID, "billing_account_id"},
		{"field_email_verified", FieldEmailVerified, "email_verified"},
		{"field_tm_create", FieldTMCreate, "tm_create"},
		{"field_tm_update", FieldTMUpdate, "tm_update"},
		{"field_tm_delete", FieldTMDelete, "tm_delete"},
		{"field_deleted", FieldDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
