package campaign

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Status       Status `json:"status"`
	ServiceLevel int    `json:"service_level"`

	// resource info
	OutplanID uuid.UUID `json:"outplan_id"`
	OutdialID uuid.UUID `json:"outdial_id"`
	QueueID   uuid.UUID `json:"queue_id"`

	NextCampaignID uuid.UUID `json:"next_campaign_id"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Campaign) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID: h.ID,

		Name:   h.Name,
		Detail: h.Detail,

		Status:       h.Status,
		ServiceLevel: h.ServiceLevel,

		OutplanID: h.OutplanID,
		OutdialID: h.OutdialID,
		QueueID:   h.QueueID,

		NextCampaignID: h.NextCampaignID,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Campaign) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
