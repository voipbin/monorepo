package streaming

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the webhook payload for VAD events
type WebhookMessage struct {
	commonidentity.Identity

	TranscribeID uuid.UUID            `json:"transcribe_id"`
	Direction    transcript.Direction `json:"direction"`
	Message      string               `json:"message,omitempty"`
	TMEvent      *time.Time           `json:"tm_event"`
}

// ConvertWebhookMessage converts a Streaming to a WebhookMessage
func (h *Streaming) ConvertWebhookMessage(message string, tmEvent *time.Time) *WebhookMessage {
	return &WebhookMessage{
		Identity:     h.Identity,
		TranscribeID: h.TranscribeID,
		Direction:    h.Direction,
		Message:      message,
		TMEvent:      tmEvent,
	}
}

// CreateWebhookEvent implements notifyhandler.WebhookMessage interface
func (h *WebhookMessage) CreateWebhookEvent() ([]byte, error) {
	m, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}

	return m, nil
}
