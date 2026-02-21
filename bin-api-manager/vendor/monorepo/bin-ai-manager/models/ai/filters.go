package ai

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for AI queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID  uuid.UUID   `filter:"customer_id"`
	Name        string      `filter:"name"`
	Detail      string      `filter:"detail"`
	EngineType  EngineType  `filter:"engine_type"`
	EngineModel EngineModel `filter:"engine_model"`
	TTSType     TTSType     `filter:"tts_type"`
	STTType     STTType     `filter:"stt_type"`
	Deleted     bool        `filter:"deleted"`
}
