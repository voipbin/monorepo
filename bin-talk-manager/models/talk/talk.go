package talk

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

// Talk represents a talk session
type Talk struct {
	commonidentity.Identity

	Type Type `json:"type" db:"type"`

	// Timestamps
	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// WebhookMessage is the webhook payload for talk events
type WebhookMessage struct {
	commonidentity.Identity

	Type     Type   `json:"type,omitempty"`
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts Talk to WebhookMessage
func (t *Talk) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: t.Identity,
		Type:     t.Type,
		TMCreate: t.TMCreate,
		TMUpdate: t.TMUpdate,
		TMDelete: t.TMDelete,
	}
}

// CreateWebhookEvent generates WebhookEvent JSON
func (t *Talk) CreateWebhookEvent() ([]byte, error) {
	e := t.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
