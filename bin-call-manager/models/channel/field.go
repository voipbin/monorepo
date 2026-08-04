package channel

// Field represents a database field name for Channel
type Field string

const (
	FieldID         Field = "id"          // id
	FieldAsteriskID Field = "asterisk_id" // asterisk_id
	FieldName       Field = "name"        // name
	FieldType       Field = "type"        // type
	FieldTech       Field = "tech"        // tech

	FieldSIPCallID    Field = "sip_call_id"   // sip_call_id
	FieldSIPTransport Field = "sip_transport" // sip_transport
	FieldSIPData      Field = "sip_data"      // sip_data

	// NOTE (plan §5 R-B): these four columns are named src_*/dst_* in
	// call_channels, while the Go struct's json tags say source_*/destination_*.
	// The column names below are the authoritative ones.
	FieldSourceName        Field = "src_name"   // src_name
	FieldSourceNumber      Field = "src_number" // src_number
	FieldDestinationName   Field = "dst_name"   // dst_name
	FieldDestinationNumber Field = "dst_number" // dst_number

	FieldState      Field = "state"       // state
	FieldData       Field = "data"        // data
	FieldStasisName Field = "stasis_name" // stasis_name
	FieldStasisData Field = "stasis_data" // stasis_data
	FieldBridgeID   Field = "bridge_id"   // bridge_id
	FieldPlaybackID Field = "playback_id" // playback_id

	FieldDialResult  Field = "dial_result"  // dial_result
	FieldHangupCause Field = "hangup_cause" // hangup_cause

	FieldDirection     Field = "direction"      // direction
	FieldMuteDirection Field = "mute_direction" // mute_direction

	FieldTMAnswer  Field = "tm_answer"  // tm_answer
	FieldTMRinging Field = "tm_ringing" // tm_ringing
	FieldTMEnd     Field = "tm_end"     // tm_end

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
