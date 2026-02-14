package allowance

// Field represents allowance field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"
	FieldAccountID  Field = "account_id"

	FieldCycleStart Field = "cycle_start"
	FieldCycleEnd   Field = "cycle_end"

	FieldTokensTotal Field = "tokens_total"
	FieldTokensUsed  Field = "tokens_used"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
