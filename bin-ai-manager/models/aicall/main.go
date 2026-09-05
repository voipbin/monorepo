package aicall

import (
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/identity"
)

// PromptSnapshot records the prompt version and final substituted text for one
// AI participant at AIcall start time.
type PromptSnapshot struct {
	AIID            uuid.UUID `json:"ai_id"`
	PromptHistoryID uuid.UUID `json:"prompt_history_id"` // zero UUID = no history recorded yet
	Prompt          string    `json:"prompt"`
	MemberID        uuid.UUID `json:"member_id"` // zero UUID for single-AI calls
}

// MetaKeyPromptSnapshots is the Metadata map key for the prompt snapshot slice.
const MetaKeyPromptSnapshots = "prompt_snapshots"

// MetaKeyAutoAuditEnabled is the Metadata map key (bool) recording whether this AICall
// should be auto-audited when it terminates. Frozen from the participating AI option(s)
// at call-creation time.
const MetaKeyAutoAuditEnabled = "auto_audit_enabled"

// MetaKeyListenTranscribeID is the Metadata map key (string, a UUID) holding the
// transcribe session this AIcall is reading while listening to a live call.
// Read by the listen-start idempotency check and by every stop path, always with
// the AIcall row already in hand -- hence Metadata rather than a column.
const MetaKeyListenTranscribeID = "listen_transcribe_id"

// MetaKeyListenOwnsTranscribe is the Metadata map key (bool) recording whether
// THIS AIcall started the transcribe session named by MetaKeyListenTranscribeID,
// as opposed to reusing one another AIcall already had running on the same call.
// Only the owner may stop it; a non-owner must never touch a session another
// listener still depends on.
const MetaKeyListenOwnsTranscribe = "listen_owns_transcribe"

// MetaKeyListenConversationID is the Metadata map key (string, a UUID) holding
// the conversation this AIcall is listening to (design
// docs/plans/2026-09-05-insight-ai-conversation-listen-design.md §5.2.1).
// Metadata rather than a column: unlike listen_call_id there is no event-driven
// sweep that needs to query by it; every reader already has the row in hand. An
// AIcall carries at most one of MetaKeyListenTranscribeID / this key.
const MetaKeyListenConversationID = "listen_conversation_id"

// AIcall define
type AIcall struct {
	identity.Identity

	AssistanceType AssistanceType `json:"assistance_type,omitempty" db:"assistance_type"`
	AssistanceID   uuid.UUID      `json:"assistance_id,omitempty" db:"assistance_id,uuid"`

	AIEngineModel ai.EngineModel `json:"ai_engine_model,omitempty" db:"ai_engine_model"`
	AITTSType     ai.TTSType     `json:"ai_tts_type,omitempty" db:"ai_tts_type"`
	AITTSVoiceID  string         `json:"ai_tts_voice_id,omitempty" db:"ai_tts_voice_id"`
	AISTTType     ai.STTType     `json:"ai_stt_type,omitempty" db:"ai_stt_type"`
	AIVADConfig        *ai.VADConfig  `json:"ai_vad_config,omitempty" db:"ai_vad_config,json"`
	AISmartTurnEnabled bool           `json:"ai_smart_turn_enabled,omitempty" db:"ai_smart_turn_enabled"`

	Parameter map[string]any `json:"parameter,omitempty" db:"parameter,json"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
	ReferenceType ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`

	ConfbridgeID  uuid.UUID `json:"confbridge_id,omitempty" db:"confbridge_id,uuid"`
	PipecatcallID   uuid.UUID `json:"pipecatcall_id,omitempty" db:"pipecatcall_id,uuid"`
	CurrentMemberID uuid.UUID `json:"current_member_id,omitempty" db:"current_member_id,uuid"`

	// ListenCallID is the live call this contact_case Insight AIcall is
	// currently listening to, or uuid.Nil when it is not listening.
	//
	// A real column rather than a Metadata key for exactly one reason: when a
	// call hangs up, EventCMCallHangup must run WHERE listen_call_id = ? to find
	// every AIcall watching it (plural -- two Cases can share one call), and
	// JSON metadata is not usefully indexable. The transcribe id and ownership
	// flag stay in Metadata precisely because they are only ever read with the
	// row already in hand.
	//
	// Deliberately NOT exposed on the webhook -- internal plumbing, same
	// treatment as Message.PipecatcallID. See docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.8.
	ListenCallID uuid.UUID `json:"listen_call_id,omitempty" db:"listen_call_id,uuid"`

	Status Status `json:"status,omitempty" db:"status"`

	STTLanguage string `json:"stt_language,omitempty" db:"stt_language"`

	Metadata map[string]any `json:"metadata,omitempty" db:"metadata,json"`

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
	ReferenceTypeContactCase  ReferenceType = "contact_case"
)

// AssistanceType defines the type of assistance entity backing an AIcall.
type AssistanceType string

// list of assistance types
const (
	AssistanceTypeAI   AssistanceType = "ai"
	AssistanceTypeTeam AssistanceType = "team"
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
