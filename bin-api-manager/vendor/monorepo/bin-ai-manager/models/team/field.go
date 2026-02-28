package team

// Field represents a database field name for type-safe updates.
type Field string

const (
	FieldID            Field = "id"
	FieldCustomerID    Field = "customer_id"
	FieldName          Field = "name"
	FieldDetail        Field = "detail"
	FieldStartMemberID Field = "start_member_id"
	FieldMembers       Field = "members"
	FieldParameter     Field = "parameter"
	FieldTMCreate      Field = "tm_create"
	FieldTMUpdate      Field = "tm_update"
	FieldTMDelete      Field = "tm_delete"
	FieldDeleted       Field = "deleted"
)
