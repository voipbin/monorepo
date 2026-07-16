package widget

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage defines the webchat widget webhook event / external response.
// DirectID is intentionally omitted — it is an internal linkage to
// bin-direct-manager and must never be exposed externally. DirectHash IS
// exposed (unlike DirectID): it is the value the embed script needs to
// authenticate anonymous visitors, mirroring the AI/Team DirectHash pattern.
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
//
// DirectHash is intentionally left empty here -- unlike bin-ai-manager's
// AI/Team resources (which return direct_hash on every GET), the webchat
// widget's direct_hash is a one-time secret embedded directly into the
// customer's public website via the embed script. Exposing it on every GET
// would mean any authenticated agent viewing the widget list/detail page
// could exfiltrate a value meant to be shown exactly once at
// creation/regeneration time (see webchat_struct_widget.rst's "AI
// Implementation Hint"). Callers that need to surface the hash (Create,
// DirectHashRegenerate) must set WebhookMessage.DirectHash explicitly after
// calling this -- see bin-api-manager's servicehandler/webchat_widget.go.
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
