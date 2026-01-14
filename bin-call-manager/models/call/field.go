package call

// Field represents a database field name for Call
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id
	FieldOwnerType  Field = "owner_type"  // owner_type
	FieldOwnerID    Field = "owner_id"    // owner_id

	FieldChannelID    Field = "channel_id"    // channel_id
	FieldBridgeID     Field = "bridge_id"     // bridge_id
	FieldFlowID       Field = "flow_id"       // flow_id
	FieldActiveflowID Field = "activeflow_id" // activeflow_id
	FieldConfbridgeID Field = "confbridge_id" // confbridge_id
	FieldType         Field = "type"          // type

	FieldMasterCallID    Field = "master_call_id"    // master_call_id
	FieldChainedCallIDs  Field = "chained_call_ids"  // chained_call_ids
	FieldRecordingID     Field = "recording_id"      // recording_id
	FieldRecordingIDs    Field = "recording_ids"     // recording_ids
	FieldExternalMediaID Field = "external_media_id" // external_media_id
	FieldGroupcallID     Field = "groupcall_id"      // groupcall_id

	FieldSource      Field = "source"      // source
	FieldDestination Field = "destination" // destination

	FieldStatus         Field = "status"           // status
	FieldData           Field = "data"             // data
	FieldAction         Field = "action"           // action
	FieldActionNextHold Field = "action_next_hold" // action_next_hold
	FieldDirection      Field = "direction"        // direction
	FieldMuteDirection  Field = "mute_direction"   // mute_direction

	FieldHangupBy     Field = "hangup_by"     // hangup_by
	FieldHangupReason Field = "hangup_reason" // hangup_reason

	FieldDialrouteID Field = "dialroute_id" // dialroute_id
	FieldDialroutes  Field = "dialroutes"   // dialroutes

	FieldTMRinging     Field = "tm_ringing"     // tm_ringing
	FieldTMProgressing Field = "tm_progressing" // tm_progressing
	FieldTMHangup      Field = "tm_hangup"      // tm_hangup

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
