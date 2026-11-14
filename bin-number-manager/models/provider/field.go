package provider

// Field represents a database field of the Provider.
type Field string

// Field constants for the Provider model
const (
	FieldProviderName Field = "provider_name"
	FieldEnabled      Field = "enabled"
	FieldPriority     Field = "priority"
	FieldConfig       Field = "config"
	FieldUpdatedBy    Field = "updated_by"
	FieldTMCreate     Field = "tm_create"
	FieldTMUpdate     Field = "tm_update"
)
