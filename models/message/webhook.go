package message

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"`

	ConversationID uuid.UUID `json:"conversation_id"`
	Status         Status    `json:"status"`

	ReferenceType conversation.ReferenceType `json:"reference_type"`
	ReferenceID   string                     `json:"reference_id"`

	SourceID string `json:"source_id"` // message source id. always user_id

	Type Type   `json:"type"`
	Data []byte `json:"message"`

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

		SourceID: h.SourceID,

		Type: h.Type,
		Data: h.Data,

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
