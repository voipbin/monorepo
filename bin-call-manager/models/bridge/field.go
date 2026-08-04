package bridge

// Field represents a database field name for Bridge
type Field string

const (
	FieldAsteriskID Field = "asterisk_id" // asterisk_id
	FieldID         Field = "id"          // id
	FieldName       Field = "name"        // name

	FieldType    Field = "type"    // type
	FieldTech    Field = "tech"    // tech
	FieldClass   Field = "class"   // class
	FieldCreator Field = "creator" // creator

	FieldVideoMode     Field = "video_mode"      // video_mode
	FieldVideoSourceID Field = "video_source_id" // video_source_id

	FieldChannelIDs Field = "channel_ids" // channel_ids

	FieldReferenceType Field = "reference_type" // reference_type
	FieldReferenceID   Field = "reference_id"   // reference_id

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
