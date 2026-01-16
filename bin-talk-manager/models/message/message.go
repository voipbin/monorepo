package message

import (
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
