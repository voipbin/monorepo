package direct

import (
	"time"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// DirectPrefix is the prefix for direct hashes
const DirectPrefix = "direct."

// Resource type constants for Direct records
const (
	ResourceTypeAI            = "ai"
	ResourceTypeAITeam        = "ai_team"
	ResourceTypeAgent         = "agent"
	ResourceTypeQueue         = "queue"
	ResourceTypeConference    = "conference"
	ResourceTypeExtension     = "extension"
	ResourceTypeWebchatWidget = "webchat_widget"
)

// Direct data model
type Direct struct {
	commonidentity.Identity

	ResourceType string    `json:"resource_type" db:"resource_type"`
	ResourceID   uuid.UUID `json:"resource_id" db:"resource_id,uuid"`
	Hash         string    `json:"hash" db:"hash"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
}
