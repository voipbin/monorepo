package tag

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name"`   // tag's name
	Detail string `json:"detail"` // tag's detail

	TMCreate *time.Time `json:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete"` // Deleted timestamp.
}

// ConvertWebhookMessage converts to the event
func (h *Tag) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:   h.Name,
		Detail: h.Detail,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Tag) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
