package account

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
		{"field_name", FieldName, "name"},
		{"field_detail", FieldDetail, "detail"},
		{"field_plan_type", FieldPlanType, "plan_type"},
		{"field_balance_credit", FieldBalanceCredit, "balance_credit"},
		{"field_balance_token", FieldBalanceToken, "balance_token"},
		{"field_payment_type", FieldPaymentType, "payment_type"},
		{"field_payment_method", FieldPaymentMethod, "payment_method"},
		{"field_tm_last_topup", FieldTmLastTopUp, "tm_last_topup"},
		{"field_tm_next_topup", FieldTmNextTopUp, "tm_next_topup"},
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
