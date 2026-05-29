package aiprompthistory

import (
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
)

// AIPromptHistory records a single historical value of an AI's init_prompt.
type AIPromptHistory struct {
	identity.Identity // ID + CustomerID

	AIID       uuid.UUID  `json:"ai_id"                 db:"ai_id,uuid"`
	Prompt     string     `json:"prompt"                db:"prompt"`
	ProposalID uuid.UUID  `json:"proposal_id,omitempty" db:"proposal_id,uuid"` // uuid.Nil for manual updates
	TMCreate   *time.Time `json:"tm_create"             db:"tm_create"`
}
