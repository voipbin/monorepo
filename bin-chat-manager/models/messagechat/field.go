package messagechat

// Field represents database column names for Messagechat
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldChatID Field = "chat_id" // chat_id

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
