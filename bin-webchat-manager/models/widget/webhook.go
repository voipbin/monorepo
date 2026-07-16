package widget

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage defines the webchat widget webhook event / external response.
// DirectID is intentionally omitted — it is an internal linkage to
// bin-direct-manager and must never be exposed externally. DirectHash IS
// exposed on every response (GET/List/Create/Update/Regenerate), mirroring
// bin-ai-manager's AI/Team DirectHash pattern: the embed script's
// data-hash value is not actually a secret in the traditional sense --
// it's embedded directly into the customer's public website HTML, so
// hiding it from GET responses only made it harder for the customer's
// own admins to retrieve it, without providing any real confidentiality
// (anyone can read it via "View Source" on the customer's site). See
// VOIP-1264.
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name"`
	Status Status `json:"status"`

	WelcomeMessage string `json:"welcome_message"`
	FlowID         string `json:"flow_id"`

	SessionIdleTimeout int `json:"session_idle_timeout"`

	ThemeConfig *ThemeConfig `json:"theme_config,omitempty"`

	DirectHash string `json:"direct_hash,omitempty"`

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

		DirectHash: h.Hash,

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
