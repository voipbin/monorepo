package billing

// Field represents billing field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldAccountID Field = "account_id"

	FieldStatus Field = "status"

	FieldReferenceType Field = "reference_type"
	FieldReferenceID   Field = "reference_id"

	FieldCostType          Field = "cost_type"
	FieldCostUnitCount     Field = "cost_unit_count"
	FieldCostTokenPerUnit  Field = "cost_token_per_unit"
	FieldCostTokenTotal    Field = "cost_token_total"
	FieldCostCreditPerUnit Field = "cost_credit_per_unit"
	FieldCostCreditTotal   Field = "cost_credit_total"

	FieldTMBillingStart Field = "tm_billing_start"
	FieldTMBillingEnd   Field = "tm_billing_end"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
