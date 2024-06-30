package chatroom

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chat"
)

// Chatroom defines chatroom
type Chatroom struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerType  OwnerType `json:"owner_type"` //
	OwnerID    uuid.UUID `json:"owner_id"`   //

	Type   Type      `json:"type"`
	ChatID uuid.UUID `json:"chat_id"`

	RoomOwnerID    uuid.UUID   `json:"room_owner_id"`   // chatroom's owner agnet id.
	ParticipantIDs []uuid.UUID `json:"participant_ids"` // list of participated agent ids

	Name   string `json:"name"`
	Detail string `json:"detail"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// OnwerType defines
type OwnerType string

// list of onwer types
const (
	OwnerTypeNone  OwnerType = ""
	OwnerTypeAgent OwnerType = "agent"
)

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
