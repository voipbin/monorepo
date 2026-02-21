package transcript

// Field represents a database field name for Transcript
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldTranscribeID Field = "transcribe_id" // transcribe_id

	FieldDirection Field = "direction" // direction
	FieldMessage   Field = "message"   // message

	FieldTMTranscript Field = "tm_transcript" // tm_transcript

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
