package mcpserver

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage is the external-facing representation of a McpServer. It
// excludes SecretCiphertext/SecretNonce/KeyVersion entirely and exposes
// HasSecret instead -- see docs/plans/2026-09-11-mcp-tool-integration-design.md
// §8.
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	URL    string `json:"url,omitempty"`
	Status Status `json:"status,omitempty"`

	AuthType     AuthType `json:"auth_type,omitempty"`
	APIKeyHeader string   `json:"api_key_header,omitempty"`

	// OAuthVendor is exposed (never the tokens) so callers can render
	// "Connected to GitHub" / "Connected to Linear" without a secondary
	// GET -- see design §4.
	OAuthVendor string `json:"oauth_vendor,omitempty"`

	HasSecret bool `json:"has_secret"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts the internal McpServer to an external
// WebhookMessage.
func (h *McpServer) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:   h.Name,
		Detail: h.Detail,

		URL:    h.URL,
		Status: h.Status,

		AuthType:     h.AuthType,
		APIKeyHeader: h.APIKeyHeader,

		OAuthVendor: h.OAuthVendor,

		HasSecret: h.HasSecret,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *McpServer) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
