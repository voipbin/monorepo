package streaming

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the customer-facing webhook payload for speech events
type WebhookMessage struct {
	commonidentity.Identity

	StreamingID  uuid.UUID           `json:"streaming_id"`
	TranscribeID uuid.UUID           `json:"transcribe_id"`
	Direction    transcript.Direction `json:"direction"`
	Message      string              `json:"message,omitempty"`
	TMEvent      *time.Time          `json:"tm_event"`

	TMCreate *time.Time `json:"tm_create"`
}

// ConvertWebhookMessage converts a Speech to the customer-facing WebhookMessage
func (h *Speech) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity:     h.Identity,
		StreamingID:  h.StreamingID,
		TranscribeID: h.TranscribeID,
		Direction:    h.Direction,
		Message:      h.Message,
		TMEvent:      h.TMEvent,
		TMCreate:     h.TMCreate,
	}
}

// CreateWebhookEvent implements notifyhandler.WebhookMessage interface
func (h *Speech) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()
	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return m, nil
}
