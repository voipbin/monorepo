package transcript

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID           uuid.UUID        `json:"id"`
	TranscribeID uuid.UUID        `json:"transcribe_id"`
	Direction    common.Direction `json:"direction"` // direction. in/out
	Message      string           `json:"message"`   // message
	TMCreate     string           `json:"tm_create"` // timestamp
}

// ConvertWebhookMessage converts to the event
func (h *Transcript) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:           h.ID,
		TranscribeID: h.TranscribeID,
		Direction:    h.Direction,
		Message:      h.Message,
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
