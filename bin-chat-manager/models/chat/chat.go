package chat

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Chat defines chat.
type Chat struct {
	commonidentity.Identity

	Type Type `json:"type"`

	RoomOwnerID    uuid.UUID   `json:"room_owner_id"`   // owned agent id
	ParticipantIDs []uuid.UUID `json:"participant_ids"` // list of participated ids(agent ids)

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
	TypeNormal Type = "normal" // 1:1 chat
	TypeGroup  Type = "group"
)
