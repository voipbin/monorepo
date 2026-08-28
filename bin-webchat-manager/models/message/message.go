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

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404/1405). It is the parent SessionID, not the message's own ID: a
// message id is stable, but it first appears in the very event that announces the message, so
// nobody can bind to it in advance. The consumption axis is the session — an agent UI or a bot
// follows one visitor conversation and wants every message of it — and Session.ID is also the
// visitor's continuity token, which makes it the natural address. Single-message retrieval stays
// available over RPC.
//
// Both webchat event types (`webchat_message_created`, `webchat_session_ended`) split into
// resource `webchat`, so this override also makes them share one address: a single
// `webchat-manager.webchat.<session-id>.#` binding catches the whole session, messages and the
// end-of-session marker alike.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *Message) EventSubscriptionID() string {
	return h.SessionID.String()
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
