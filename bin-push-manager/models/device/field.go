package device

// Field represents a database field name for Device model
type Field string

// List of Device fields for database operations
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldAgentID  Field = "agent_id"
	FieldPlatform Field = "platform"
	FieldToken    Field = "token"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
)
