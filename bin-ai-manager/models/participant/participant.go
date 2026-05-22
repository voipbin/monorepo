package participant

import (
	"time"

	"github.com/gofrs/uuid"
)

// Participant is a single row from ai_aicall_participants.
// UUID fields use the ,uuid db tag so commondatabasehandler.ScanRow converts BINARY(16) correctly.
// Participant deliberately does NOT embed identity.Identity — it is a composite-key join row
// (no separate id, no customer_id) over a table created by PR #934.
type Participant struct {
	AIID     uuid.UUID  `json:"ai_id"     db:"ai_id,uuid"`
	AIcallID uuid.UUID  `json:"aicall_id" db:"aicall_id,uuid"`
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}

// WebhookMessage is the external-facing representation.
type WebhookMessage struct {
	AIID     uuid.UUID  `json:"ai_id"`
	AIcallID uuid.UUID  `json:"aicall_id"`
	TMCreate *time.Time `json:"tm_create"`
}

// ConvertWebhookMessage converts a Participant to its external representation.
func (p *Participant) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		AIID:     p.AIID,
		AIcallID: p.AIcallID,
		TMCreate: p.TMCreate,
	}
}
