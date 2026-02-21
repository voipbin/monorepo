package accesskey

// Field represents a database field name for Accesskey
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldTokenHash   Field = "token_hash"
	FieldTokenPrefix Field = "token_prefix"

	FieldTMExpire Field = "tm_expire"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
