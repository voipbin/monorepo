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

	FieldPlanType   Field = "plan_type"
	FieldPlanStatus Field = "plan_status"

	FieldBalanceCredit Field = "balance_credit"
	FieldBalanceToken  Field = "balance_token"

	FieldPaymentType   Field = "payment_type"
	FieldPaymentMethod Field = "payment_method"

	FieldPaddleSubscriptionID Field = "paddle_subscription_id"
	FieldPaddleCustomerID     Field = "paddle_customer_id"

	FieldTmLastTopUp Field = "tm_last_topup"
	FieldTmNextTopUp Field = "tm_next_topup"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
