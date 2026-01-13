package chatroom

// Field represents database column names for Chatroom
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id
	FieldOwnerType  Field = "owner_type"  // owner_type
	FieldOwnerID    Field = "owner_id"    // owner_id

	FieldType   Field = "type"    // type
	FieldChatID Field = "chat_id" // chat_id

	FieldRoomOwnerID    Field = "room_owner_id"   // room_owner_id
	FieldParticipantIDs Field = "participant_ids" // participant_ids

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
