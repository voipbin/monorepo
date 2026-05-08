package provider

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"`

	Type     Type   `json:"type"`
	Hostname string `json:"hostname"` // destination

	// sip type techs
	TechPrefix  string            `json:"tech_prefix"`  // tech prefix. valid only for the sip type.
	TechPostfix string            `json:"tech_postfix"` // tech postfix. valid only for the sip type.
	TechHeaders map[string]string `json:"tech_headers"` // tech headers. valid only for the sip type.

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Metadata map[string]interface{} `json:"metadata"`

	Codecs string `json:"codecs"`

	// health check
	HealthStatus    string     `json:"health_status"`
	HealthCheckedAt *time.Time `json:"health_checked_at"`

	// timestamp
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Provider) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID: h.ID,

		Type:     h.Type,
		Hostname: h.Hostname,

		TechPrefix:  h.TechPrefix,
		TechPostfix: h.TechPostfix,
		TechHeaders: h.TechHeaders,

		Name:   h.Name,
		Detail: h.Detail,

		Metadata: h.Metadata,

		Codecs: h.Codecs,

		HealthStatus:    h.HealthStatus,
		HealthCheckedAt: h.HealthCheckedAt,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Provider) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
