package session

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Session is the platform's existing CHILD resource in the direct-token
// model (ai/aicall's exact shape). Session.ID doubles as the visitor's
// identity — it is what the widget round-trips as a continuity token, and
// what the WebSocket topic is scoped by. No separate VisitorID field
// exists.
type Session struct {
	commonidentity.Identity // ID, CustomerID — Identity.ID IS the visitor's continuity token

	WidgetID uuid.UUID `json:"widget_id,omitempty" db:"widget_id,uuid"`
	Status   Status    `json:"status,omitempty" db:"status"`

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`

	// WelcomeMessage is a transient, DB-unbound field: an in-memory
	// copy of the owning Widget.WelcomeMessage attached only to the
	// Create response (POST /webchat_sessions). List/Get/End responses
	// always leave this empty -- see design doc §6. db:"-" is an
	// established pattern in this codebase (bin-agent-manager's
	// Addresses field).
	WelcomeMessage string `json:"welcome_message,omitempty" db:"-"`

	TMLastActivity *time.Time `json:"tm_last_activity,omitempty" db:"tm_last_activity"`
	TMCreate       *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate       *time.Time `json:"tm_update,omitempty" db:"tm_update"`
	TMEnd          *time.Time `json:"tm_end,omitempty" db:"tm_end"` // lifecycle marker, distinct from tm_delete
	TMDelete       *time.Time `json:"-" db:"tm_delete"`             // standard soft-delete sentinel
}

// Status type
type Status string

// list of statuses
//
// (created on first inbound message, no session_id supplied by client) --> active
// active --(idle_timeout elapsed, or explicit close)--------------------> ended
//
// ended is terminal; a subsequent message with no session_id (or an ended
// one) creates a NEW Session — i.e., a new conversation, with its own new
// unguessable ID.
const (
	StatusActive Status = "active"
	StatusEnded  Status = "ended"
)
