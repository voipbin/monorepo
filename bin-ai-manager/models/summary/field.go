package summary

// Field represents Summary field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldActiveflowID Field = "activeflow_id"
	FieldOnEndFlowID  Field = "on_end_flow_id"

	FieldReferenceType Field = "reference_type"
	FieldReferenceID   Field = "reference_id"

	FieldStatus   Field = "status"
	FieldLanguage Field = "language"
	FieldContent  Field = "content"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
