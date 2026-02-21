package chat

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Type represents the type of chat
type Type string

// Chat type constants
const (
	// TypeDirect - 1:1 Direct Message (Private between two users)
	TypeDirect Type = "direct"

	// TypeGroup - Group Direct Message (Private multi-user chat, invite-only)
	TypeGroup Type = "group"

	// TypeTalk - Public Open Channel (Topic-based, searchable, e.g., #general, #random)
	// Mapped to "talk" to distinguish from VoIP channels
	TypeTalk Type = "talk"
)

// Chat represents a chat session
type Chat struct {
	commonidentity.Identity

	Type   Type   `json:"type" db:"type"`
	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	// Member count (atomically updated on participant join/leave)
	MemberCount int `json:"member_count" db:"member_count"`

	// Timestamps
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// WebhookMessage is the webhook payload for chat events
type WebhookMessage struct {
	commonidentity.Identity

	Type        Type       `json:"type,omitempty"`
	Name        string     `json:"name,omitempty"`
	Detail      string     `json:"detail,omitempty"`
	MemberCount int        `json:"member_count,omitempty"`
	TMCreate    *time.Time `json:"tm_create"`
	TMUpdate    *time.Time `json:"tm_update"`
	TMDelete    *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts Chat to WebhookMessage
func (t *Chat) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity:    t.Identity,
		Type:        t.Type,
		Name:        t.Name,
		Detail:      t.Detail,
		MemberCount: t.MemberCount,
		TMCreate:    t.TMCreate,
		TMUpdate:    t.TMUpdate,
		TMDelete:    t.TMDelete,
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
