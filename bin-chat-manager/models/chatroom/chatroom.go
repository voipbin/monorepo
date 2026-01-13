package chatroom

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chat"
)

// Chatroom defines chatroom
type Chatroom struct {
	commonidentity.Identity
	commonidentity.Owner

	Type   Type      `json:"type" db:"type"`
	ChatID uuid.UUID `json:"chat_id" db:"chat_id,uuid"`

	RoomOwnerID    uuid.UUID   `json:"room_owner_id" db:"room_owner_id,uuid"`     // chatroom's owner agnet id.
	ParticipantIDs []uuid.UUID `json:"participant_ids" db:"participant_ids,json"` // list of participated agent ids

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
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
