package participant

import (
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of a Participant.
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
