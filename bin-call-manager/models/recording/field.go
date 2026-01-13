package recording

// Field represents a database field name for Recording
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id
	FieldOwnerType  Field = "owner_type"  // owner_type
	FieldOwnerID    Field = "owner_id"    // owner_id

	FieldActiveflowID  Field = "activeflow_id"   // activeflow_id
	FieldReferenceType Field = "reference_type"  // reference_type
	FieldReferenceID   Field = "reference_id"    // reference_id
	FieldStatus        Field = "status"          // status
	FieldFormat        Field = "format"          // format
	FieldOnEndFlowID   Field = "on_end_flow_id"  // on_end_flow_id
	FieldRecordingName Field = "recording_name"  // recording_name
	FieldFilenames     Field = "filenames"       // filenames
	FieldAsteriskID    Field = "asterisk_id"     // asterisk_id
	FieldChannelIDs    Field = "channel_ids"     // channel_ids

	FieldTMStart  Field = "tm_start"  // tm_start
	FieldTMEnd    Field = "tm_end"    // tm_end
	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
