package aicall

// Field represents AIcall field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldAssistanceType Field = "assistance_type"
	FieldAssistanceID   Field = "assistance_id"

	FieldAIEngineModel Field = "ai_engine_model"
	FieldAITTSType     Field = "ai_tts_type"
	FieldAITTSVoiceID  Field = "ai_tts_voice_id"
	FieldAISTTType     Field = "ai_stt_type"
	FieldAIVADConfig   Field = "ai_vad_config"

	FieldParameter Field = "parameter"

	FieldActiveflowID  Field = "activeflow_id"
	FieldReferenceType Field = "reference_type"
	FieldReferenceID   Field = "reference_id"

	FieldConfbridgeID  Field = "confbridge_id"
	FieldPipecatcallID Field = "pipecatcall_id"

	FieldStatus Field = "status"

	FieldGender   Field = "gender"
	FieldLanguage Field = "language"

	FieldTMEnd    Field = "tm_end"
	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
