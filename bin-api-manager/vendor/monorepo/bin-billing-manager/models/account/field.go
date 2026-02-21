package account

// Field represents account field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldStatus Field = "status"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldPlanType Field = "plan_type"

	FieldBalanceCredit Field = "balance_credit"
	FieldBalanceToken  Field = "balance_token"

	FieldPaymentType   Field = "payment_type"
	FieldPaymentMethod Field = "payment_method"

	FieldTmLastTopUp Field = "tm_last_topup"
	FieldTmNextTopUp Field = "tm_next_topup"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
