package transcribe

// Field represents a database field name for Transcribe
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldActiveflowID Field = "activeflow_id"  // activeflow_id
	FieldOnEndFlowID  Field = "on_end_flow_id" // on_end_flow_id

	FieldReferenceType Field = "reference_type" // reference_type
	FieldReferenceID   Field = "reference_id"   // reference_id

	FieldStatus    Field = "status"    // status
	FieldHostID    Field = "host_id"   // host_id
	FieldLanguage  Field = "language"  // language
	FieldDirection Field = "direction" // direction

	FieldStreamingIDs Field = "streaming_ids" // streaming_ids

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
