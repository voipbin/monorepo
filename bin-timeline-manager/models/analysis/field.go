package analysis

// Field represents an Analysis column for database queries.
type Field string

const (
	FieldID           Field = "id"
	FieldCustomerID   Field = "customer_id"
	FieldActiveflowID Field = "activeflow_id"

	FieldStatus Field = "status"
	FieldResult Field = "result"
	FieldModel  Field = "model"
	FieldError  Field = "error"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// FieldDeleted is a synthetic filter (maps to a tm_delete predicate).
	FieldDeleted Field = "deleted"
)
