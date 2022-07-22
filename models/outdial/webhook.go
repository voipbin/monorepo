package outdial

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	CampaignID uuid.UUID `json:"campaign_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Data string `json:"data"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Outdial) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		CampaignID: h.CampaignID,

		Name:   h.Name,
		Detail: h.Detail,

		Data: h.Data,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Outdial) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
