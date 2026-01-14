package email

// Field represents a database field name for Email model
type Field string

// List of Email fields for database operations
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldActiveflowID Field = "activeflow_id"

	FieldProviderType        Field = "provider_type"
	FieldProviderReferenceID Field = "provider_reference_id"

	FieldSource       Field = "source"
	FieldDestinations Field = "destinations"

	FieldStatus  Field = "status"
	FieldSubject Field = "subject"
	FieldContent Field = "content"

	FieldAttachments Field = "attachments"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
