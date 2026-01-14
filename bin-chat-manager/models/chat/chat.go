package chat

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Chat defines chat.
type Chat struct {
	commonidentity.Identity

	Type Type `json:"type" db:"type"`

	RoomOwnerID    uuid.UUID   `json:"room_owner_id" db:"room_owner_id,uuid"`     // owned agent id
	ParticipantIDs []uuid.UUID `json:"participant_ids" db:"participant_ids,json"` // list of participated ids(agent ids)

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
	TypeNormal Type = "normal" // 1:1 chat
	TypeGroup  Type = "group"
)
