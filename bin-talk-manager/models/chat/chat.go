package chat

import (
	"encoding/json"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Type constants
const (
	TypeNormal = "normal"
	TypeGroup  = "group"
)

type Type string

// Chat represents a chat session
type Chat struct {
	commonidentity.Identity

	Type Type `json:"type" db:"type"`

	// Timestamps
	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// WebhookMessage is the webhook payload for chat events
type WebhookMessage struct {
	commonidentity.Identity

	Type     Type   `json:"type,omitempty"`
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts Chat to WebhookMessage
func (t *Chat) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: t.Identity,
		Type:     t.Type,
		TMCreate: t.TMCreate,
		TMUpdate: t.TMUpdate,
		TMDelete: t.TMDelete,
	}
}

// CreateWebhookEvent generates WebhookEvent JSON
func (t *Chat) CreateWebhookEvent() ([]byte, error) {
	e := t.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
