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
		{"field_cost_type", FieldCostType, "cost_type"},
		{"field_cost_unit_count", FieldCostUnitCount, "cost_unit_count"},
		{"field_cost_token_per_unit", FieldCostTokenPerUnit, "cost_token_per_unit"},
		{"field_cost_token_total", FieldCostTokenTotal, "cost_token_total"},
		{"field_cost_credit_per_unit", FieldCostCreditPerUnit, "cost_credit_per_unit"},
		{"field_cost_credit_total", FieldCostCreditTotal, "cost_credit_total"},
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
