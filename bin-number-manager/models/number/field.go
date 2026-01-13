package number

// Field represents a database field for Number
type Field string

// Field constants for Number model
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldNumber Field = "number"

	FieldCallFlowID    Field = "call_flow_id"
	FieldMessageFlowID Field = "message_flow_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldProviderName        Field = "provider_name"
	FieldProviderReferenceID Field = "provider_reference_id"

	FieldStatus Field = "status"

	FieldT38Enabled       Field = "t38_enabled"
	FieldEmergencyEnabled Field = "emergency_enabled"

	FieldTMPurchase Field = "tm_purchase"
	FieldTMRenew    Field = "tm_renew"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
