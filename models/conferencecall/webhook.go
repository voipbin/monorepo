package conferencecall

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines conferencecall webhook event
type WebhookMessage struct {
	ID           uuid.UUID `json:"id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	ConferenceID uuid.UUID `json:"conference_id"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Status Status `json:"status"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Conferencecall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:           h.ID,
		CustomerID:   h.CustomerID,
		ConferenceID: h.ConferenceID,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Status: h.Status,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Conferencecall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
