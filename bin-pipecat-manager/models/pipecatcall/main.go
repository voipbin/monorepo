package pipecatcall

import (
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type Pipecatcall struct {
	identity.Identity

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	HostID string `json:"host_id,omitempty"`

	LLMType     LLMType          `json:"llm_type,omitempty"`
	LLMMessages []map[string]any `json:"llm_messages,omitempty"`
	STTType     STTType          `json:"stt_type,omitempty"`
	TTSType     TTSType          `json:"tts_type,omitempty"`
	TTSVoiceID  string           `json:"tts_voice_id,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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
