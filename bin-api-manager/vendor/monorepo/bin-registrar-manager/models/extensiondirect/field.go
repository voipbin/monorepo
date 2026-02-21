package extensiondirect

// Field represents a typed field name for extension direct queries
type Field string

// Field constants for extension direct
const (
	FieldID          Field = "id"
	FieldCustomerID  Field = "customer_id"
	FieldExtensionID Field = "extension_id"
	FieldHash        Field = "hash"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
