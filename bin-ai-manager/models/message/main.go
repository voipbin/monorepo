package message

import (
	"time"

	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Message defines
type Message struct {
	identity.Identity

	AIcallID     uuid.UUID `json:"aicall_id,omitempty" db:"aicall_id,uuid"`
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
	ActiveAIID   uuid.UUID `json:"active_ai_id,omitempty" db:"active_ai_id,uuid"`

	Direction Direction `json:"direction,omitempty" db:"direction"`
	Role      Role      `json:"role,omitempty" db:"role"`
	Content   string    `json:"content,omitempty" db:"content"`

	ToolCalls  []ToolCall `json:"tool_calls,omitempty" db:"tool_calls,json"`
	ToolCallID string     `json:"tool_call_id,omitempty" db:"tool_call_id"`

	// Origin marks whether this message was the AI's own initiative rather than
	// a reply to, or a question from, somebody. Empty for every ordinary
	// message. See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md
	// §5.6.2.
	Origin Origin `json:"origin,omitempty" db:"origin"`

	PipecatcallID  uuid.UUID      `json:"-" db:"pipecatcall_id,uuid"`
	DeliveryStatus DeliveryStatus `json:"-" db:"delivery_status"`

	// InReplyToMessageID is the ID of the user-authored message this assistant
	// message answers. Populated on assistant messages created via
	// EventPMMessageBotLLM (echoed from bin-pipecat-manager's
	// pmmessage.Message.InReplyToMessageID) to disambiguate which inbound
	// message triggered this response when an AIcall is reused for a rapid
	// sequence of sends (e.g. an agent asking a second question before the
	// first bot response arrives). Zero UUID for user messages and for
	// assistant messages where no correlation was available. See VOIP-1234
	// design doc §4-1.
	InReplyToMessageID uuid.UUID `json:"in_reply_to_message_id,omitempty" db:"in_reply_to_message_id,uuid"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404 / VOIP-1405 §2.3). It is the parent AIcallID, not the message's
// own ID: a message id is stable and persisted, but it first becomes known to a subscriber in the
// `aimessage_created` event itself, so nobody can bind to it in advance. Every real consumption
// pattern follows the AIcall conversation, so the AIcall is the address. Single-message retrieval
// stays available over RPC.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *Message) EventSubscriptionID() string {
	return h.AIcallID.String()
}

// Role defiens
type Role string

// list of roles
const (
	RoleNone      Role = ""
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleFunction  Role = "function"
	RoleTool         Role = "tool"
	RoleNotification Role = "notification"
)

type Direction string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
	DirectionNone     Direction = ""
)

// Origin marks how a message came to exist, orthogonally to Role.
//
// It is a string enum rather than a bool, matching Role / Direction /
// DeliveryStatus, so a future third origin does not need another column.
type Origin string

// list of origins
const (
	// OriginNone is the default: every message that answers or asks something.
	OriginNone Origin = ""

	// OriginProactive marks an AI-initiated notification -- something the AI
	// chose to say without being asked, via the notify_agent tool while
	// listening to a live call. It is REAL conversational content: it is stored
	// as role=assistant, it is replayed into future LLM context (so the AI
	// remembers what it told the agent when they ask "what did you mean?"), and
	// the frontends render it with a distinct treatment.
	OriginProactive Origin = "proactive"

	// OriginListenInternal marks the mechanical tool-call and tool-result rows
	// written during a listen evaluation turn. These are NEVER replayed into any
	// future context -- getPipecatcallMessages excludes them at the SQL layer.
	//
	// Without that exclusion they would accumulate (up to 2 rows per turn, up to
	// the per-AIcall turn cap) and push the AIcall's own system prompt and the
	// agent's real Q&A history out of the capped replay window the next time the
	// agent asks a question. See the design doc §5.4.5.
	OriginListenInternal Origin = "listen_internal"
)

// DeliveryStatus tracks whether a message has been successfully delivered
// to the user (e.g. TTS audio actually played out to the call).
type DeliveryStatus string

// list of delivery statuses
const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
)
