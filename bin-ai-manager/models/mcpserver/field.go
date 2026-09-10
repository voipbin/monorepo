package mcpserver

// Field represents a database field name for type-safe updates.
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldURL    Field = "url"
	FieldStatus Field = "status"

	FieldAuthType     Field = "auth_type"
	FieldAPIKeyHeader Field = "api_key_header"

	FieldSecretCiphertext Field = "secret_ciphertext"
	FieldSecretNonce      Field = "secret_nonce"
	FieldKeyVersion       Field = "key_version"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"
	FieldDeleted  Field = "deleted"
)
