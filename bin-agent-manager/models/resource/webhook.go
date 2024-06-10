package resource

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID            uuid.UUID     `json:"id"`
	CustomerID    uuid.UUID     `json:"customer_id"`
	OwnerID       uuid.UUID     `json:"owner_id"`
	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Data interface{} `json:"data"`

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// ConvertWebhookMessage converts to the event
func (h *Resource) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:            h.ID,
		CustomerID:    h.CustomerID,
		OwnerID:       h.OwnerID,
		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Data: h.Data,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Resource) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
