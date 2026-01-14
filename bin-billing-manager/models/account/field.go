package account

// Field represents account field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldType Field = "type"

	FieldBalance Field = "balance"

	FieldPaymentType   Field = "payment_type"
	FieldPaymentMethod Field = "payment_method"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
