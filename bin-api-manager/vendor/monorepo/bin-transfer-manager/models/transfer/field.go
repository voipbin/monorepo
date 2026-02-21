package transfer

// Field represents a database field name for Transfer model
type Field string

// List of Transfer fields for database operations
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldType Field = "type"

	FieldTransfererCallID    Field = "transferer_call_id"
	FieldTransfereeAddresses Field = "transferee_addresses"
	FieldTransfereeCallID    Field = "transferee_call_id"

	FieldGroupcallID  Field = "groupcall_id"
	FieldConfbridgeID Field = "confbridge_id"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
