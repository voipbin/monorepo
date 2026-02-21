package aicall

import (
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/identity"
)

// AIcall define
type AIcall struct {
	identity.Identity

	AIID          uuid.UUID      `json:"ai_id,omitempty" db:"ai_id,uuid"`
	AIEngineType  ai.EngineType  `json:"ai_engine_type,omitempty" db:"ai_engine_type"`
	AIEngineModel ai.EngineModel `json:"ai_engine_model,omitempty" db:"ai_engine_model"`
	AIEngineData  map[string]any `json:"ai_engine_data,omitempty" db:"ai_engine_data,json"`
	AITTSType     ai.TTSType     `json:"ai_tts_type,omitempty" db:"ai_tts_type"`
	AITTSVoiceID  string         `json:"ai_tts_voice_id,omitempty" db:"ai_tts_voice_id"`
	AISTTType     ai.STTType     `json:"ai_stt_type,omitempty" db:"ai_stt_type"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
	ReferenceType ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`

	ConfbridgeID  uuid.UUID `json:"confbridge_id,omitempty" db:"confbridge_id,uuid"`
	PipecatcallID uuid.UUID `json:"pipecatcall_id,omitempty" db:"pipecatcall_id,uuid"`

	Status Status `json:"status,omitempty" db:"status"`

	Gender   Gender `json:"gender,omitempty" db:"gender"`
	Language string `json:"language,omitempty" db:"language"`

	TMEnd    *time.Time `json:"tm_end" db:"tm_end"`
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone         ReferenceType = ""
	ReferenceTypeCall         ReferenceType = "call"
	ReferenceTypeConversation ReferenceType = "conversation"
	ReferenceTypeTask         ReferenceType = "task"
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
	GenderNone    Gender = ""
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderNeutral Gender = "neutral"
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
