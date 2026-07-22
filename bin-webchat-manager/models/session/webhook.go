package session

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the webchat session webhook event / external response.
type WebhookMessage struct {
	commonidentity.Identity

	WidgetID uuid.UUID `json:"widget_id"`
	Status   Status    `json:"status"`
	PageURL  string    `json:"page_url,omitempty"`

	TMLastActivity *time.Time `json:"tm_last_activity,omitempty"`
	TMCreate       *time.Time `json:"tm_create,omitempty"`
	TMUpdate       *time.Time `json:"tm_update,omitempty"`
	TMEnd          *time.Time `json:"tm_end,omitempty"`
}

// ConvertWebhookMessage converts the Session into a WebhookMessage.
func (h *Session) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		WidgetID: h.WidgetID,
		Status:   h.Status,
		PageURL:  h.PageURL,

		TMLastActivity: h.TMLastActivity,
		TMCreate:       h.TMCreate,
		TMUpdate:       h.TMUpdate,
		TMEnd:          h.TMEnd,
	}
}

// CreateWebhookEvent generates the WebhookEvent payload.
func (h *Session) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
