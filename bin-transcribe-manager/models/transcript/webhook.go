package transcript

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	TranscribeID uuid.UUID `json:"transcribe_id"`
	Direction    Direction `json:"direction"` // direction. in/out
	Message      string    `json:"message"`   // message

	TMTranscript string `json:"tm_transcript"`
	TMCreate     string `json:"tm_create"` // timestamp
}

// ConvertWebhookMessage converts to the event
func (h *Transcript) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		TranscribeID: h.TranscribeID,
		Direction:    h.Direction,
		Message:      h.Message,
		TMTranscript: h.TMTranscript,
		TMCreate:     h.TMCreate,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Transcript) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
