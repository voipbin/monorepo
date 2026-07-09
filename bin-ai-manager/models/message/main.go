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

// DeliveryStatus tracks whether a message has been successfully delivered
// to the user (e.g. TTS audio actually played out to the call).
type DeliveryStatus string

// list of delivery statuses
const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
)
