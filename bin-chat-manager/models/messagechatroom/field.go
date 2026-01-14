package messagechatroom

// Field represents database column names for Messagechatroom
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id
	FieldOwnerType  Field = "owner_type"  // owner_type
	FieldOwnerID    Field = "owner_id"    // owner_id

	FieldChatroomID    Field = "chatroom_id"    // chatroom_id
	FieldMessagechatID Field = "messagechat_id" // messagechat_id

	FieldSource Field = "source" // source
	FieldType   Field = "type"   // type
	FieldText   Field = "text"   // text
	FieldMedias Field = "medias" // medias

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
