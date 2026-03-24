package direct

// Field represents a database field name for Direct model
type Field string

// List of Direct fields for database operations
const (
	FieldID           Field = "id"
	FieldCustomerID   Field = "customer_id"
	FieldResourceType Field = "resource_type"
	FieldResourceID   Field = "resource_id"
	FieldHash         Field = "hash"
	FieldTMCreate     Field = "tm_create"
	FieldTMUpdate     Field = "tm_update"
)
