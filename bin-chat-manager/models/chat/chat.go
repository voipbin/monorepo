package chat

import (
	"github.com/gofrs/uuid"
)

// Chat defines chat.
type Chat struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Type Type `json:"type"`

	OwnerID        uuid.UUID   `json:"owner_id"`        // owned agent id
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
