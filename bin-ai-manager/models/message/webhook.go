package message

import (
	"encoding/json"
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type WebhookMessage struct {
	identity.Identity
	ChatbotcallID uuid.UUID `json:"chatbotcall_id,omitempty"`

	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	Direction Direction `json:"direction"`

	TMCreate string `json:"tm_create,omitempty"`
}

func (h *Message) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity:      h.Identity,
		ChatbotcallID: h.ChatbotcallID,

		Role:      h.Role,
		Content:   h.Content,
		Direction: h.Direction,
		TMCreate:  h.TMCreate,
	}
}

func (h *Message) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
