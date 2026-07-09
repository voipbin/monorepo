package ai

// Field represents AI field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"
	FieldType   Field = "type"

	FieldEngineModel Field = "engine_model"
	FieldParameter   Field = "parameter"
	FieldEngineKey   Field = "engine_key"
	FieldRagID       Field = "rag_id"

	FieldInitPrompt Field = "init_prompt"

	FieldCurrentPromptHistoryID Field = "current_prompt_history_id"

	FieldTTSType    Field = "tts_type"
	FieldTTSVoiceID Field = "tts_voice_id"

	FieldSTTType          Field = "stt_type"
	FieldSTTLanguage      Field = "stt_language"
	FieldVADConfig        Field = "vad_config"
	FieldSmartTurnEnabled Field = "smart_turn_enabled"

	FieldAutoAICallAuditEnabled Field = "auto_aicall_audit_enabled"

	FieldToolNames Field = "tool_names"

	FieldDirectID   Field = "direct_id"
	FieldDirectHash Field = "direct_hash"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
