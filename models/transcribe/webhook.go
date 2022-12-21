package transcribe

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`          // Transcribe id
	CustomerID uuid.UUID `json:"customer_id"` // customer

	ReferenceType ReferenceType `json:"reference_type"` // reference's type
	ReferenceID   uuid.UUID     `json:"reference_id"`   // call/conference/recording's id

	Status   Status    `json:"status"`
	HostID   uuid.UUID `json:"host_id"`  // host id
	Language string    `json:"language"` // BCP47 type's language code. en-US

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Transcribe) ConvertWebhookMessage() *WebhookMessage {

	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Status:   h.Status,
		HostID:   h.HostID,
		Language: h.Language,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Transcribe) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
