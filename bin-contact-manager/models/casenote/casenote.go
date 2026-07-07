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

// AuthorType constants.
const (
	AuthorTypeAgent  = "agent"
	AuthorTypeSystem = "system"
)
