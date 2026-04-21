package providercall

// Field represents database field names for ProviderCall
type Field string

const (
	FieldID Field = "id"

	FieldCustomerID   Field = "customer_id"
	FieldProviderID   Field = "provider_id"
	FieldFlowID       Field = "flow_id"
	FieldSource       Field = "source"
	FieldDestinations Field = "destinations"
	FieldAnonymous    Field = "anonymous"

	FieldCallIDs      Field = "call_ids"
	FieldGroupcallIDs Field = "groupcall_ids"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
