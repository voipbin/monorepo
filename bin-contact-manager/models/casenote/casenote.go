package casenote

import (
	"time"

	"github.com/gofrs/uuid"
)

// CaseNote is an internal, agent-facing annotation on a Case (design
// §3.5). It is physically and transport-isolated from customer-facing
// data: never surfaced in any customer webhook or API response, and its
// creation event (case_note_created) MUST be published via the plain
// notifyHandler.PublishEvent() primitive -- never PublishWebhookEvent().
type CaseNote struct {
	ID         uuid.UUID `json:"id"          db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`
	CaseID     uuid.UUID `json:"case_id"     db:"case_id,uuid"`

	AuthorType string     `json:"author_type" db:"author_type"`
	AuthorID   *uuid.UUID `json:"author_id"   db:"author_id,uuid"`

	Text string `json:"text" db:"text"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404 §4.2): the parent CaseID, NOT the note's own ID.
//
// The note id is stable and persisted, so this is a deliberate Category B choice (design §2.1):
// the note id only becomes knowable through the created event itself, so pre-binding to it has no
// value, while every real consumption pattern follows the case. Adopting the case address
// forfeits own-id instance subscription for notes -- accepted; single-note retrieval stays
// available over RPC. It also keeps case_note_created and case_note_deleted (see
// CaseNoteDeletedEvent) in the same key space as the case tag/contact events, so one
// `contact-manager.case.<case-id>.#` binding follows the whole case.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *CaseNote) EventSubscriptionID() string {
	return h.CaseID.String()
}

// AuthorType constants.
const (
	AuthorTypeAgent  = "agent"
	AuthorTypeSystem = "system"
)
