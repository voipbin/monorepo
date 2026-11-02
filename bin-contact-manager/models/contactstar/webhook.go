package contactstar

import (
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the externally visible shape of a star: the contact the
// calling agent starred (or un-starred) and when. Internal ids (the row id,
// customer id and agent id) are not exposed.
type WebhookMessage struct {
	ContactID uuid.UUID  `json:"contact_id"`
	Starred   bool       `json:"starred"`
	TMCreate  *time.Time `json:"tm_create,omitempty"`
}

// ConvertWebhookMessage returns the externally visible shape of a star
// that exists, so Starred is always true.
func (h *ContactStar) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ContactID: h.ContactID,
		Starred:   true,
		TMCreate:  h.TMCreate,
	}
}
