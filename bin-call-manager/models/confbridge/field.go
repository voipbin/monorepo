package confbridge

// Field represents a database field name for Confbridge
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldActiveflowID  Field = "activeflow_id"  // activeflow_id
	FieldReferenceType Field = "reference_type" // reference_type
	FieldReferenceID   Field = "reference_id"   // reference_id

	FieldType     Field = "type"      // type
	FieldStatus   Field = "status"    // status
	FieldBridgeID Field = "bridge_id" // bridge_id
	FieldFlags    Field = "flags"     // flags

	FieldChannelCallIDs Field = "channel_call_ids" // channel_call_ids

	FieldRecordingID  Field = "recording_id"  // recording_id
	FieldRecordingIDs Field = "recording_ids" // recording_ids

	FieldExternalMediaID Field = "external_media_id" // external_media_id

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
