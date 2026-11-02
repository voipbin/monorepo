package pin

import (
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-talk-manager/models/message"
)

// TargetType is what a pin points to.
type TargetType string

// Target types
const (
	TargetTypeChat    TargetType = "chat"
	TargetTypeMessage TargetType = "message"
)

// IsValid returns true if the target type is one of the known types.
func (t TargetType) IsValid() bool {
	return t == TargetTypeChat || t == TargetTypeMessage
}

// Pin is a personal state of an owner (an agent): the owner pinned a chat or a message.
type Pin struct {
	commonidentity.Identity
	commonidentity.Owner

	// ChatID is the chat of the target. For a chat pin it equals TargetID; for a message pin it is
	// the chat of the message. It is filled by the server.
	ChatID     uuid.UUID  `json:"chat_id" db:"chat_id,uuid"`
	TargetType TargetType `json:"target_type" db:"target_type"`
	TargetID   uuid.UUID  `json:"target_id" db:"target_id,uuid"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`

	// Message is the pinned message of a message pin. It is not a column; it is filled by the list.
	Message *message.Message `json:"message,omitempty"`
}

// WebhookMessage is the externally exposed shape of a pin.
type WebhookMessage struct {
	ID         uuid.UUID               `json:"id"`
	TargetType TargetType              `json:"target_type"`
	TargetID   uuid.UUID               `json:"target_id"`
	ChatID     uuid.UUID               `json:"chat_id"`
	Message    *message.WebhookMessage `json:"message,omitempty"`
	TMCreate   *time.Time              `json:"tm_create"`
}

// ConvertWebhookMessage converts Pin to WebhookMessage.
func (p *Pin) ConvertWebhookMessage() (*WebhookMessage, error) {
	res := &WebhookMessage{
		ID:         p.ID,
		TargetType: p.TargetType,
		TargetID:   p.TargetID,
		ChatID:     p.ChatID,
		TMCreate:   p.TMCreate,
	}

	if p.Message != nil {
		m, err := p.Message.ConvertWebhookMessage()
		if err != nil {
			return nil, err
		}
		res.Message = m
	}

	return res, nil
}
