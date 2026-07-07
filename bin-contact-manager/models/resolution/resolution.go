package resolution

import (
	"time"

	"github.com/gofrs/uuid"
)

// Resolution records a manual attribution (positive) or suppression (negative)
// of a contact_interaction, OR (contact-case-management design §3.3) a whole
// contact_case, to a contact. It is append-only with soft-delete retraction:
// correction = tm_delete (retract existing) + new row (re-attribute).
//
// Exactly one of InteractionID/CaseID is set on any given row (never both,
// never neither) -- interaction-level Resolutions (the original, far more
// common shape) set InteractionID and leave CaseID nil; case-level
// Resolutions (new) set CaseID and leave InteractionID nil. See VOIP-1204
// §3.3 and VOIP-1209 for full design context.
type Resolution struct {
	ID             uuid.UUID  `json:"id"               db:"id,uuid"`
	CustomerID     uuid.UUID  `json:"customer_id"      db:"customer_id,uuid"`
	ContactID      uuid.UUID  `json:"contact_id"       db:"contact_id,uuid"`
	InteractionID  *uuid.UUID `json:"interaction_id"   db:"interaction_id,uuid"`
	CaseID         *uuid.UUID `json:"case_id"          db:"case_id,uuid"`
	ResolutionType string     `json:"resolution_type"  db:"resolution_type"`
	ResolvedByType string     `json:"resolved_by_type" db:"resolved_by_type"`
	ResolvedByID   uuid.UUID  `json:"resolved_by_id"   db:"resolved_by_id,uuid"`
	TMCreate       *time.Time `json:"tm_create"        db:"tm_create"`
	TMUpdate       *time.Time `json:"tm_update"        db:"tm_update"`
	TMDelete       *time.Time `json:"tm_delete"        db:"tm_delete"`
}

// Resolution type constants.
const (
	// ResolutionTypePositive attaches an interaction to a contact (manual
	// attribution that automatic peer-matching missed).
	ResolutionTypePositive = "positive"

	// ResolutionTypeNegative suppresses an interaction from a contact (removes
	// a wrong automatic match). Negative always wins over positive for the same
	// (contact_id, interaction_id) — set-MINUS semantics, never LWW.
	ResolutionTypeNegative = "negative"
)

// ResolvedBy type constants.
const (
	ResolvedByTypeAgent  = "agent"
	ResolvedByTypeSystem = "system"
	ResolvedByTypeRule   = "rule"
)

// ResolvedByIDSystem is the canonical resolved_by_id for system/rule
// resolutions where no agent is involved.
var ResolvedByIDSystem = uuid.Nil
