package contact

// Field represents a database field name for Contact model
type Field string

// List of Contact fields for database operations
const (
	// Identity fields
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	// Basic info fields
	FieldFirstName   Field = "first_name"
	FieldLastName    Field = "last_name"
	FieldDisplayName Field = "display_name"
	FieldCompany     Field = "company"
	FieldJobTitle    Field = "job_title"

	// Tracking fields
	FieldSource     Field = "source"
	FieldExternalID Field = "external_id"
	FieldNotes      Field = "notes"

	// Timestamp fields
	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// Virtual field for soft delete filtering
	FieldDeleted Field = "deleted"
)
