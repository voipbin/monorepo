package participant

import (
	"encoding/json"
	"time"

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
	TMJoined *time.Time `json:"tm_joined" db:"tm_joined"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404/1405 §2.3). It is the parent ChatID, not the participant's own
// ID.
//
// The participant id is stable and persisted, so this is a Category B override: the id first
// appears in the participant's own `chatparticipant_added` event, which means nobody can pre-bind
// to it, while every real consumption pattern follows a chat. Addressing by the parent makes
// `talk-manager.chatparticipant.<chat-id>.#` deliver the roster changes of that conversation.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (p *Participant) EventSubscriptionID() string {
	return p.ChatID.String()
}

// WebhookMessage is the webhook payload for participant events
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID   uuid.UUID  `json:"chat_id"`
	TMJoined *time.Time `json:"tm_joined"`
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
