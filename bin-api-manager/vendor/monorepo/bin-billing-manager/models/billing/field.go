package billing

// Field represents billing field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldAccountID Field = "account_id"

	FieldTransactionType Field = "transaction_type"
	FieldStatus          Field = "status"

	FieldReferenceType Field = "reference_type"
	FieldReferenceID   Field = "reference_id"

	FieldCostType      Field = "cost_type"
	FieldUsageDuration Field = "usage_duration"
	FieldBillableUnits Field = "billable_units"

	FieldRateTokenPerUnit  Field = "rate_token_per_unit"
	FieldRateCreditPerUnit Field = "rate_credit_per_unit"

	FieldAmountToken  Field = "amount_token"
	FieldAmountCredit Field = "amount_credit"

	FieldBalanceTokenSnapshot  Field = "balance_token_snapshot"
	FieldBalanceCreditSnapshot Field = "balance_credit_snapshot"

	FieldIdempotencyKey Field = "idempotency_key"

	FieldTMBillingStart Field = "tm_billing_start"
	FieldTMBillingEnd   Field = "tm_billing_end"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
