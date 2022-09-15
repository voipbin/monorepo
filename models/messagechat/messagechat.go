package messagechat

import (
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/message"
)

// Messagechat defines message for the chat
type Messagechat struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	ChatID uuid.UUID `json:"chat_id"`

	Message message.Message `json:"message"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
