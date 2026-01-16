package message

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Type constants
const (
	TypeNormal = "normal"
	TypeSystem = "system"
)

type Type string

// Message represents a talk message
type Message struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID   uuid.UUID  `json:"chat_id" db:"chat_id,uuid"`
	ParentID *uuid.UUID `json:"parent_id,omitempty" db:"parent_id,uuid"`

	Type     Type   `json:"type" db:"type"`
	Text     string `json:"text" db:"text"`
	Medias   string `json:"medias" db:"medias"`     // JSON string
	Metadata string `json:"metadata" db:"metadata"` // JSON string

	// Timestamps
	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// WebhookMessage is the webhook payload for message events
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID   uuid.UUID  `json:"chat_id"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`

	Type     Type     `json:"type"`
	Text     string   `json:"text"`
	Medias   []Media  `json:"medias"`
	Metadata Metadata `json:"metadata"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Media represents a media attachment (simplified from chat-manager)
type Media struct {
	Type string `json:"type"` // "file", "link", "address", "agent"
	// Add specific fields as needed based on Type
}

// ConvertWebhookMessage converts Message to WebhookMessage
func (m *Message) ConvertWebhookMessage() (*WebhookMessage, error) {
	// Parse Medias JSON string to []Media
	var medias []Media
	if m.Medias != "" {
		if err := json.Unmarshal([]byte(m.Medias), &medias); err != nil {
			return nil, err
		}
	}

	// Parse Metadata JSON string to Metadata struct
	var metadata Metadata
	if m.Metadata != "" {
		if err := json.Unmarshal([]byte(m.Metadata), &metadata); err != nil {
			return nil, err
		}
	}

	return &WebhookMessage{
		Identity: m.Identity,
		Owner:    m.Owner,
		ChatID:   m.ChatID,
		ParentID: m.ParentID,
		Type:     m.Type,
		Text:     m.Text,
		Medias:   medias,
		Metadata: metadata,
		TMCreate: m.TMCreate,
		TMUpdate: m.TMUpdate,
		TMDelete: m.TMDelete,
	}, nil
}

// CreateWebhookEvent generates WebhookEvent JSON
func (m *Message) CreateWebhookEvent() ([]byte, error) {
	e, err := m.ConvertWebhookMessage()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return data, nil
}
