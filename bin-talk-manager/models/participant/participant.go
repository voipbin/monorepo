package participant

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// ParticipantInput is used for adding participants during chat creation
type ParticipantInput struct {
	OwnerType string    `json:"owner_type"`
	OwnerID   uuid.UUID `json:"owner_id"`
}

// Participant represents a chat participant
type Participant struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID uuid.UUID `json:"chat_id" db:"chat_id,uuid"`

	// Timestamps
	TMJoined string `json:"tm_joined" db:"tm_joined"`
}

// WebhookMessage is the webhook payload for participant events
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID   uuid.UUID `json:"chat_id"`
	TMJoined string    `json:"tm_joined"`
}

// ConvertWebhookMessage converts Participant to WebhookMessage
func (p *Participant) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: p.Identity,
		Owner:    p.Owner,
		ChatID:   p.ChatID,
		TMJoined: p.TMJoined,
	}
}

// CreateWebhookEvent generates WebhookEvent JSON
func (p *Participant) CreateWebhookEvent() ([]byte, error) {
	e := p.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
