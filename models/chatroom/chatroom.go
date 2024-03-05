package chatroom

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
)

// Chatroom defines chatroom
type Chatroom struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	AgentID    uuid.UUID `json:"agent_id"`

	Type   Type      `json:"type"`
	ChatID uuid.UUID `json:"chat_id"`

	OwnerID        uuid.UUID   `json:"onwer_id"`        // chatroom's owner agnet id.
	ParticipantIDs []uuid.UUID `json:"participant_ids"` // list of participated agent ids

	Name   string `json:"name"`
	Detail string `json:"detail"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type define
type Type string

// list of types
const (
	TypeUnkonwn Type = "unknown"
	TypeNormal  Type = "normal"
	TypeGroup   Type = "group"
)

// ConvertType converts the chat's type to the chatroom's type.
func ConvertType(chatType chat.Type) Type {
	switch chatType {
	case chat.TypeNormal:
		return TypeNormal

	case chat.TypeGroup:
		return TypeGroup

	default:
		return TypeUnkonwn
	}
}
