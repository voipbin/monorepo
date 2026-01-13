package conference

// Field represents a database field name for conference
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldConfbridgeID Field = "confbridge_id" // confbridge_id
	FieldType         Field = "type"          // type

	FieldStatus Field = "status" // status

	FieldName    Field = "name"    // name
	FieldDetail  Field = "detail"  // detail
	FieldData    Field = "data"    // data
	FieldTimeout Field = "timeout" // timeout

	FieldPreFlowID  Field = "pre_flow_id"  // pre_flow_id
	FieldPostFlowID Field = "post_flow_id" // post_flow_id

	FieldConferencecallIDs Field = "conferencecall_ids" // conferencecall_ids

	FieldRecordingID  Field = "recording_id"  // recording_id
	FieldRecordingIDs Field = "recording_ids" // recording_ids

	FieldTranscribeID  Field = "transcribe_id"  // transcribe_id
	FieldTranscribeIDs Field = "transcribe_ids" // transcribe_ids

	FieldTMEnd Field = "tm_end" // tm_end

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
