package number

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID     uuid.UUID `json:"id"`
	Number string    `json:"number"`
	FlowID uuid.UUID `json:"flow_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Status Status `json:"status"`

	T38Enabled       bool `json:"t38_enabled"`
	EmergencyEnabled bool `json:"emergency_enabled"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Number) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:     h.ID,
		Number: h.Number,
		FlowID: h.FlowID,

		Name:   h.Name,
		Detail: h.Detail,

		Status: h.Status,

		T38Enabled:       h.T38Enabled,
		EmergencyEnabled: h.EmergencyEnabled,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Number) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
