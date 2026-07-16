package message

// Field represents database field names for Message
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldSessionID Field = "session_id"
	FieldDirection Field = "direction"
	FieldStatus    Field = "status"

	FieldSenderID     Field = "sender_id"
	FieldActiveflowID Field = "activeflow_id"

	FieldText Field = "text"

	FieldTMCreate Field = "tm_create"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
