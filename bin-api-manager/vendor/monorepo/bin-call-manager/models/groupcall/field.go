package groupcall

// Field represents a database field name for Groupcall
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id
	FieldOwnerType  Field = "owner_type"  // owner_type
	FieldOwnerID    Field = "owner_id"    // owner_id

	FieldStatus Field = "status"  // status
	FieldFlowID Field = "flow_id" // flow_id

	FieldSource       Field = "source"       // source
	FieldDestinations Field = "destinations" // destinations

	FieldMasterCallID      Field = "master_call_id"      // master_call_id
	FieldMasterGroupcallID Field = "master_groupcall_id" // master_groupcall_id

	FieldRingMethod   Field = "ring_method"   // ring_method
	FieldAnswerMethod Field = "answer_method" // answer_method

	FieldAnswerCallID Field = "answer_call_id" // answer_call_id
	FieldCallIDs      Field = "call_ids"       // call_ids

	FieldAnswerGroupcallID Field = "answer_groupcall_id" // answer_groupcall_id
	FieldGroupcallIDs      Field = "groupcall_ids"       // groupcall_ids

	FieldCallCount      Field = "call_count"      // call_count
	FieldGroupcallCount Field = "groupcall_count" // groupcall_count
	FieldDialIndex      Field = "dial_index"      // dial_index

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
