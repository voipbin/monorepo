package participant

import (
	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Participant represents a talk participant
type Participant struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID uuid.UUID `json:"chat_id" db:"chat_id,uuid"`

	// Timestamps
	TMJoined string `json:"tm_joined" db:"tm_joined"`
}

// WebhookMessage is the webhook payload for participant events
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	ChatID   uuid.UUID `json:"chat_id"`
	TMJoined string    `json:"tm_joined"`
}
