package message

// Field represents a database field name for Message model
type Field string

// List of Message fields for database operations
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldType Field = "type"

	FieldSource  Field = "source"
	FieldTargets Field = "targets"

	FieldProviderName        Field = "provider_name"
	FieldProviderReferenceID Field = "provider_reference_id"

	FieldText      Field = "text"
	FieldMedias    Field = "medias"
	FieldDirection Field = "direction"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
