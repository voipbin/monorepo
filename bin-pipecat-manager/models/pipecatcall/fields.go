package pipecatcall

type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldActiveflowID  Field = "activeflow_id"  // activeflow_id
	FieldReferenceType Field = "reference_type" // reference_type
	FieldReferenceID   Field = "reference_id"   // reference_id

	FieldHostID Field = "host_id" // host_id

	FieldLLMType     Field = "llm_type"     // llm_type
	FieldLLMMessages Field = "llm_messages" // llm_messages
	FieldSTTType     Field = "stt_type"     // stt_type
	FieldTTSType     Field = "tts_type"     // tts_type
	FieldTTSVoiceID  Field = "tts_voice_id" // tts_voice_id

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
