package contactstar

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// ContactStar marks a contact as starred by one agent. The star is private
// to the agent: no other agent sees it. The row is a hard-delete record
// (un-starring removes it), so it has no tm_update or tm_delete.
type ContactStar struct {
	commonidentity.Identity

	AgentID   uuid.UUID `json:"agent_id"   db:"agent_id,uuid"`
	ContactID uuid.UUID `json:"contact_id" db:"contact_id,uuid"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}
