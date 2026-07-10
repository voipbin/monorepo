package interaction

import (
	"time"

	"github.com/gofrs/uuid"
)

// Interaction is an immutable append-only fact in the CRM interaction timeline.
// It records that a channel-level event (call or conversation message) touched a
// particular remote endpoint (peer). Identity resolution (peer -> contact_id) is
// done at read time (VOIP-1209), not projection time.
type Interaction struct {
	ID         uuid.UUID `json:"id"          db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id"  db:"customer_id,uuid"`

	// Direction of the interaction from our perspective ("incoming"/"outgoing").
	Direction string `json:"direction" db:"direction"`

	// Remote endpoint (the peer's address — match key for read-time contact resolution).
	// peer_target is stored normalized via commonaddress.NormalizeTarget so it is
	// bit-identical to contact_addresses.target.
	PeerType   string `json:"peer_type"   db:"peer_type"`
	PeerTarget string `json:"peer_target" db:"peer_target"`

	// Our local endpoint (for attribution: which number/account received/sent).
	// Not in the idempotency unique; not indexed (attribution only).
	LocalType   string `json:"local_type"   db:"local_type"`
	LocalTarget string `json:"local_target" db:"local_target"`

	// Origin channel record. State and body are fetched at read time via
	// (reference_type, reference_id).
	ReferenceType string    `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID `json:"reference_id"   db:"reference_id,uuid"`

	// TMInteraction is the origin event time, used for display sort.
	// Nullable: may be nil when the origin event omits TMCreate (e.g. call events
	// with omitempty). Stored as NULL in that case — do not dereference.
	TMInteraction *time.Time `json:"tm_interaction" db:"tm_interaction"`

	// TMCreate is the projection insert time, used as the pagination cursor.
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}
