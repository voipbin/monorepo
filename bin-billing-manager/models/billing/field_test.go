package billing

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
		{"field_account_id", FieldAccountID, "account_id"},
		{"field_status", FieldStatus, "status"},
		{"field_reference_type", FieldReferenceType, "reference_type"},
		{"field_reference_id", FieldReferenceID, "reference_id"},
		{"field_cost_per_unit", FieldCostPerUnit, "cost_per_unit"},
		{"field_cost_total", FieldCostTotal, "cost_total"},
		{"field_billing_unit_count", FieldBillingUnitCount, "billing_unit_count"},
		{"field_tm_billing_start", FieldTMBillingStart, "tm_billing_start"},
		{"field_tm_billing_end", FieldTMBillingEnd, "tm_billing_end"},
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
