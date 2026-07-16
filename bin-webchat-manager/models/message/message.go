package message

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Message is a single chat message within a webchat Session.
type Message struct {
	commonidentity.Identity

	// WidgetID is denormalized from Session.WidgetID onto every Message
	// so downstream event consumers (conversation-manager's §16
	// message-manager-pattern integration) can build Conversation.Self
	// without a second RPC back to webchat-manager to resolve
	// Session -> Widget. Session.ID remains the sole visitor identity
	// (Conversation.Peer); WidgetID here is purely a denormalized
	// convenience field for the event payload, not a second identity.
	WidgetID uuid.UUID `json:"widget_id,omitempty" db:"widget_id,uuid"`

	SessionID uuid.UUID `json:"session_id,omitempty" db:"session_id,uuid"`
	Direction Direction `json:"direction,omitempty" db:"direction"`
	Status    Status    `json:"status,omitempty" db:"status"`

	// SenderID: agent user ID for an agent-typed outbound reply; empty for
	// flow/AI-originated or inbound messages. Always an Agent ID when set,
	// never a visitor identity — visitors are identified by SessionID, not
	// by a SenderID on their own messages.
	SenderID uuid.UUID `json:"sender_id,omitempty" db:"sender_id,uuid"`

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`

	Text string `json:"text,omitempty" db:"text"`

	TMCreate *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMDelete *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}

// Direction type
type Direction string

// list of directions
const (
	DirectionInbound  Direction = "inbound"  // visitor -> VoIPbin
	DirectionOutbound Direction = "outbound" // VoIPbin -> visitor
)

// Status type
type Status string

// list of statuses
const (
	StatusSent      Status = "sent"
	StatusDelivered Status = "delivered" // best-effort
	StatusFailed    Status = "failed"    // event publish itself failed (rare; RabbitMQ down)
)
