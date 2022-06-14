package message

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"`

	ConversationID uuid.UUID `json:"conversation_id"`
	Status         Status    `json:"status"`

	ReferenceType conversation.ReferenceType `json:"reference_type"`
	ReferenceID   string                     `json:"reference_id"`

	SourceTarget string `json:"source_target"`

	Text   string        `json:"text"`
	Medias []media.Media `json:"medias"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Message) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID: h.ID,

		ConversationID: h.ConversationID,
		Status:         h.Status,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		SourceTarget: h.SourceTarget,

		Text:   h.Text,
		Medias: h.Medias,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Message) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
