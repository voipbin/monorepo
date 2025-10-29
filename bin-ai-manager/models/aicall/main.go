package aicall

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/identity"
)

// AIcall define
type AIcall struct {
	identity.Identity

	AIID          uuid.UUID      `json:"ai_id,omitempty"`
	AIEngineType  ai.EngineType  `json:"ai_engine_type,omitempty"`
	AIEngineModel ai.EngineModel `json:"ai_engine_model,omitempty"`
	AIEngineData  map[string]any `json:"ai_engine_data,omitempty"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	ConfbridgeID  uuid.UUID `json:"confbridge_id,omitempty"`
	TranscribeID  uuid.UUID `json:"transcribe_id,omitempty"`
	PipecatcallID uuid.UUID `json:"pipecatcall_id,omitempty"`

	Status Status `json:"status,omitempty"`

	Gender   Gender `json:"gender,omitempty"`
	Language string `json:"language,omitempty"`

	// tts streaming info
	TTSStreamingID    uuid.UUID `json:"tts_streaming_id,omitempty"`     // TTS streaming ID
	TTSStreamingPodID string    `json:"tts_streaming_pod_id,omitempty"` // TTS's pod ID for streaming.

	TMEnd    string `json:"tm_end,omitempty"`
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone         ReferenceType = ""
	ReferenceTypeCall         ReferenceType = "call"
	ReferenceTypeConversation ReferenceType = "conversation"
)

// Status define
type Status string

// list of Statuses
const (
	StatusInitiating  Status = "initiating"
	StatusProgressing Status = "progressing"
	StatusPausing     Status = "pausing"
	StatusResuming    Status = "resuming"
	StatusTerminating Status = "terminating" // the call is terminating.
	StatusTerminated  Status = "terminated"
)

// Gender define
type Gender string

// list of genders
const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderNuetral Gender = "neutral"
)

// Message defines
type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

// MessageRole defiens
type MessageRole string

// list of roles
const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleFunction  MessageRole = "function"
	MessageRoleTool      MessageRole = "tool"
)
