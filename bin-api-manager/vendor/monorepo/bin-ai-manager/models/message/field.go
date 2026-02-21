package message

// Field represents Message field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldAIcallID Field = "aicall_id"

	FieldDirection Field = "direction"
	FieldRole      Field = "role"
	FieldContent   Field = "content"

	FieldToolCalls  Field = "tool_calls"
	FieldToolCallID Field = "tool_call_id"

	FieldTMCreate Field = "tm_create"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
