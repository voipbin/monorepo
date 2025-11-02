package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
)

// V1DataAIsPost is
// v1 data type request struct for
// /v1/ais POST
type V1DataAIsPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Detail     string    `json:"detail,omitempty"`

	EngineType  ai.EngineType  `json:"engine_type,omitempty"`
	EngineModel ai.EngineModel `json:"engine_model,omitempty"`
	EngineData  map[string]any `json:"engine_data,omitempty"`
	EngineKey   string         `json:"engine_key,omitempty"`

	InitPrompt string `json:"init_prompt,omitempty"`

	TTSType    ai.TTSType `json:"tts_type,omitempty"`
	TTSVoiceID string     `json:"tts_voice_id,omitempty"`

	STTType ai.STTType `json:"stt_type,omitempty"`
}

// V1DataAIsIDPut is
// v1 data type request struct for
// /v1/ais/<ai-id> PUT
type V1DataAIsIDPut struct {
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	EngineType  ai.EngineType  `json:"engine_type,omitempty"`
	EngineModel ai.EngineModel `json:"engine_model,omitempty"`
	EngineData  map[string]any `json:"engine_data,omitempty"`
	EngineKey   string         `json:"engine_key,omitempty"`

	InitPrompt string `json:"init_prompt,omitempty"`

	TTSType    ai.TTSType `json:"tts_type,omitempty"`
	TTSVoiceID string     `json:"tts_voice_id,omitempty"`

	STTType ai.STTType `json:"stt_type,omitempty"`
}
