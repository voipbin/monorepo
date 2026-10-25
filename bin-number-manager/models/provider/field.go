package provider

// Field represents a database field of the Provider.
type Field string

// Field constants for the Provider model
const (
	FieldProviderName     Field = "provider_name"
	FieldEnabled          Field = "enabled"
	FieldPriority         Field = "priority"
	FieldConfig           Field = "config"
	FieldSecretCiphertext Field = "secret_ciphertext"
	FieldKeyVersion       Field = "key_version"
	FieldUpdatedBy        Field = "updated_by"
	FieldSecretUpdatedBy  Field = "secret_updated_by"
	FieldTMSecretUpdate   Field = "tm_secret_update"
	FieldTMCreate         Field = "tm_create"
	FieldTMUpdate         Field = "tm_update"
)
