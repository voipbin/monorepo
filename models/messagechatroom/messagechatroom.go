package messagechatroom

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/message"
)

// Messagechatroom defines the message for the chatroom
type Messagechatroom struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	ChatroomID uuid.UUID `json:"chatroom_id"`

	MessagechatID uuid.UUID `json:"messagechat_id"`

	Message message.Message `json:"message"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
