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

	FieldCostPerUnit Field = "cost_per_unit"
	FieldCostTotal   Field = "cost_total"

	FieldBillingUnitCount Field = "billing_unit_count"

	FieldTMBillingStart Field = "tm_billing_start"
	FieldTMBillingEnd   Field = "tm_billing_end"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
