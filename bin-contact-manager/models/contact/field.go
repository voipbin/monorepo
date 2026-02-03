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

// PhoneNumberField represents a database field name for PhoneNumber model
type PhoneNumberField string

// List of PhoneNumber fields for database operations
const (
	PhoneNumberFieldID         PhoneNumberField = "id"
	PhoneNumberFieldCustomerID PhoneNumberField = "customer_id"
	PhoneNumberFieldContactID  PhoneNumberField = "contact_id"
	PhoneNumberFieldNumber     PhoneNumberField = "number"
	PhoneNumberFieldNumberE164 PhoneNumberField = "number_e164"
	PhoneNumberFieldType       PhoneNumberField = "type"
	PhoneNumberFieldIsPrimary  PhoneNumberField = "is_primary"
	PhoneNumberFieldTMCreate   PhoneNumberField = "tm_create"
)

// EmailField represents a database field name for Email model
type EmailField string

// List of Email fields for database operations
const (
	EmailFieldID         EmailField = "id"
	EmailFieldCustomerID EmailField = "customer_id"
	EmailFieldContactID  EmailField = "contact_id"
	EmailFieldAddress    EmailField = "address"
	EmailFieldType       EmailField = "type"
	EmailFieldIsPrimary  EmailField = "is_primary"
	EmailFieldTMCreate   EmailField = "tm_create"
)
