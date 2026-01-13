package pipecatcall

import (
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type Pipecatcall struct {
	identity.Identity

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
	ReferenceType ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`

	HostID string `json:"host_id,omitempty" db:"host_id"`

	LLMType     LLMType          `json:"llm_type,omitempty" db:"llm_type"`
	LLMMessages []map[string]any `json:"llm_messages,omitempty" db:"llm_messages,json"`

	STTType     STTType `json:"stt_type,omitempty" db:"stt_type"`
	STTLanguage string  `json:"stt_language,omitempty" db:"stt_language"`

	TTSType     TTSType `json:"tts_type,omitempty" db:"tts_type"`
	TTSLanguage string  `json:"tts_language,omitempty" db:"tts_language"`
	TTSVoiceID  string  `json:"tts_voice_id,omitempty" db:"tts_voice_id"`

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}

type ReferenceType string

const (
	ReferenceTypeCall   ReferenceType = "call"
	ReferenceTypeAICall ReferenceType = "ai_call"
)

// LLMType
// consist of (vendor) + . + (model)
// e.g. openai.gpt-4, anthropic.claude-2
type LLMType string

type STTType string

const (
	STTTypeNone     STTType = ""
	STTTypeDeepgram STTType = "deepgram"
)

type TTSType string

const (
	TTSTypeNone       TTSType = ""
	TTSTypeCartesia   TTSType = "cartesia"
	TTSTypeElevenLabs TTSType = "elevenlabs"
)
