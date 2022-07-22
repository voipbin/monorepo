package extension

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	DomainID uuid.UUID `json:"domain_id"`

	Extension string `json:"extension"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Extension) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID: h.ID,
		CustomerID: h.CustomerID,

		Name:     h.Name,
		Detail:   h.Detail,
		DomainID: h.DomainID,

		Extension: h.Extension,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Extension) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
