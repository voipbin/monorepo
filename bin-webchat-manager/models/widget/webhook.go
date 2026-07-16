package widget

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage defines the webchat widget webhook event / external response.
// DirectID is intentionally omitted — it is an internal linkage to
// bin-direct-manager and must never be exposed externally.
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name"`
	Status Status `json:"status"`

	WelcomeMessage string `json:"welcome_message"`
	FlowID         string `json:"flow_id"`

	SessionIdleTimeout int `json:"session_idle_timeout"`

	ThemeConfig *ThemeConfig `json:"theme_config,omitempty"`

	TMCreate *time.Time `json:"tm_create,omitempty"`
	TMUpdate *time.Time `json:"tm_update,omitempty"`
	TMDelete *time.Time `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts the Widget into a WebhookMessage.
func (h *Widget) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:   h.Name,
		Status: h.Status,

		WelcomeMessage: h.WelcomeMessage,
		FlowID:         h.FlowID.String(),

		SessionIdleTimeout: h.SessionIdleTimeout,

		ThemeConfig: h.ThemeConfig,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent payload.
func (h *Widget) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
