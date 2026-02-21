package message

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/media"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	ConversationID uuid.UUID `json:"conversation_id,omitempty"`
	Direction      Direction `json:"direction,omitempty"`
	Status         Status    `json:"status,omitempty"`

	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	Text   string        `json:"text,omitempty"`
	Medias []media.Media `json:"medias,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Message) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ConversationID: h.ConversationID,
		Direction:      h.Direction,
		Status:         h.Status,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

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
