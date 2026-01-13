package customer

// Field represents a database field name for Customer
type Field string

const (
	FieldID     Field = "id"
	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldEmail       Field = "email"
	FieldPhoneNumber Field = "phone_number"
	FieldAddress     Field = "address"

	FieldWebhookMethod Field = "webhook_method"
	FieldWebhookURI    Field = "webhook_uri"

	FieldBillingAccountID Field = "billing_account_id"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
