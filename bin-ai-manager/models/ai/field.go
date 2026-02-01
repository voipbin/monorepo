package ai

// Field represents AI field for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldEngineType  Field = "engine_type"
	FieldEngineModel Field = "engine_model"
	FieldEngineData  Field = "engine_data"
	FieldEngineKey   Field = "engine_key"

	FieldInitPrompt Field = "init_prompt"

	FieldTTSType    Field = "tts_type"
	FieldTTSVoiceID Field = "tts_voice_id"

	FieldSTTType Field = "stt_type"

	FieldToolNames Field = "tool_names"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
