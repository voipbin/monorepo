package mention

import (
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Mention is the index row of one agent mention in one message. It is written together with the
// message and is never exposed directly; the mention list returns the mentioning messages.
type Mention struct {
	commonidentity.Identity

	ChatID    uuid.UUID `json:"chat_id" db:"chat_id,uuid"`
	MessageID uuid.UUID `json:"message_id" db:"message_id,uuid"`
	AgentID   uuid.UUID `json:"agent_id" db:"agent_id,uuid"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}
