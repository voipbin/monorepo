package messagechatroom

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/message"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	ChatroomID uuid.UUID `json:"chatroom_id"`

	MessagechatID uuid.UUID `json:"messagechat_id"`

	Message message.Message `json:"message"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Messagechatroom) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		ChatroomID: h.ChatroomID,

		MessagechatID: h.MessagechatID,

		Message: h.Message,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Messagechatroom) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
