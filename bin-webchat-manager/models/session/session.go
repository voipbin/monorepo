package session

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
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

	// PageURL is the page the widget was embedded on when this Session was
	// created, captured client-side from window.location.href at
	// POST /webchat_sessions time. Best-effort: absent for pre-upgrade embed
	// snippets and for sessions created via the admin/accesskey direct-create
	// path (no browser page exists in that path). NEVER re-captured on
	// mid-session navigation -- this is a session-creation-time fact, exactly
	// like WidgetID.
	PageURL string `json:"page_url,omitempty" db:"page_url"`

	// Referrer is document.referrer at session-creation time -- the page the
	// visitor was on immediately before arriving at the page that embeds the
	// widget. Distinct from PageURL (the page the widget is currently
	// embedded on). Best-effort/absent under the same conditions as PageURL.
	Referrer string `json:"referrer,omitempty" db:"referrer"`

	// Peer/Local mirror the pattern shipped on kase.Case/interaction.Interaction,
	// but Peer uses the new commonaddress.TypeWebSession (not TypeWebchat), so
	// Peer is type-distinguishable from Local. Both are ALWAYS PRESENT in JSON
	// output (no omitempty) -- computable unconditionally at Create() time.
	Peer  commonaddress.Address `json:"peer" db:"peer,json"`
	Local commonaddress.Address `json:"local" db:"local,json"`

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`

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
