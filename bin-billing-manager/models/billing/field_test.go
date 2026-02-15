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
		{"field_transaction_type", FieldTransactionType, "transaction_type"},
		{"field_status", FieldStatus, "status"},
		{"field_reference_type", FieldReferenceType, "reference_type"},
		{"field_reference_id", FieldReferenceID, "reference_id"},
		{"field_cost_type", FieldCostType, "cost_type"},
		{"field_usage_duration", FieldUsageDuration, "usage_duration"},
		{"field_billable_units", FieldBillableUnits, "billable_units"},
		{"field_rate_token_per_unit", FieldRateTokenPerUnit, "rate_token_per_unit"},
		{"field_rate_credit_per_unit", FieldRateCreditPerUnit, "rate_credit_per_unit"},
		{"field_amount_token", FieldAmountToken, "amount_token"},
		{"field_amount_credit", FieldAmountCredit, "amount_credit"},
		{"field_balance_token_snapshot", FieldBalanceTokenSnapshot, "balance_token_snapshot"},
		{"field_balance_credit_snapshot", FieldBalanceCreditSnapshot, "balance_credit_snapshot"},
		{"field_idempotency_key", FieldIdempotencyKey, "idempotency_key"},
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
