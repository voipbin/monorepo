package chat

import (
	"encoding/json"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-talk-manager/models/participant"
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

	// Timestamps
	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`

	// Participants in this chat (populated for list operations)
	Participants []*participant.Participant `json:"participants,omitempty" db:"-"`
}

// WebhookMessage is the webhook payload for chat events
type WebhookMessage struct {
	commonidentity.Identity

	Type     Type   `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Detail   string `json:"detail,omitempty"`
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`

	// Participants in this chat
	Participants []*participant.WebhookMessage `json:"participants,omitempty"`
}

// ConvertWebhookMessage converts Chat to WebhookMessage
func (t *Chat) ConvertWebhookMessage() *WebhookMessage {
	wm := &WebhookMessage{
		Identity: t.Identity,
		Type:     t.Type,
		Name:     t.Name,
		Detail:   t.Detail,
		TMCreate: t.TMCreate,
		TMUpdate: t.TMUpdate,
		TMDelete: t.TMDelete,
	}

	// Convert participants if present
	if len(t.Participants) > 0 {
		wm.Participants = make([]*participant.WebhookMessage, 0, len(t.Participants))
		for _, p := range t.Participants {
			wm.Participants = append(wm.Participants, p.ConvertWebhookMessage())
		}
	}

	return wm
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
