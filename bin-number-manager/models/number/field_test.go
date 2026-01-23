package number

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
		{"field_customer_id", FieldCustomerID, "customer_id"},
		{"field_number", FieldNumber, "number"},
		{"field_call_flow_id", FieldCallFlowID, "call_flow_id"},
		{"field_message_flow_id", FieldMessageFlowID, "message_flow_id"},
		{"field_name", FieldName, "name"},
		{"field_detail", FieldDetail, "detail"},
		{"field_provider_name", FieldProviderName, "provider_name"},
		{"field_provider_reference_id", FieldProviderReferenceID, "provider_reference_id"},
		{"field_status", FieldStatus, "status"},
		{"field_t38_enabled", FieldT38Enabled, "t38_enabled"},
		{"field_emergency_enabled", FieldEmergencyEnabled, "emergency_enabled"},
		{"field_tm_purchase", FieldTMPurchase, "tm_purchase"},
		{"field_tm_renew", FieldTMRenew, "tm_renew"},
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
