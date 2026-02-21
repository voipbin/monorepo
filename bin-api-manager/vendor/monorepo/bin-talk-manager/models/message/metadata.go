package message

import (
	"time"

	"github.com/gofrs/uuid"
)

// Metadata represents message metadata (reactions, etc.)
type Metadata struct {
	Reactions []Reaction `json:"reactions"`
}

// Reaction represents a single emoji reaction
type Reaction struct {
	Emoji     string     `json:"emoji"`
	OwnerType string     `json:"owner_type"`
	OwnerID   uuid.UUID  `json:"owner_id"`
	TMCreate  *time.Time `json:"tm_create"`
}
