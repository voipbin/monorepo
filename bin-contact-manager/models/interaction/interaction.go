package interaction

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

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

	// Peer is the remote endpoint (match key for read-time contact
	// resolution), stored as JSON. Peer.Target is normalized via
	// commonaddress.NormalizeTarget so it is bit-identical to
	// contact_addresses.target.
	Peer commonaddress.Address `json:"peer" db:"peer,json"`

	// Local is our own endpoint (attribution: which number/account
	// received/sent), stored as JSON. Not in the idempotency unique; not
	// separately indexed (attribution only). ALWAYS PRESENT in JSON
	// output (no `omitempty` -- see kase.Case's §4.1 note, which applies
	// identically here: Go's omitempty has no effect on non-pointer
	// struct fields). A zero Local (historical pre-adb8daac2bb0 rows, or
	// any future event with no known local endpoint) serializes as
	// `"local":{}`; the underlying MySQL column may independently be
	// SQL NULL (migrated historical rows, §3.2) or JSON `'{}'` (any row
	// written by this design's Go code with a zero self) -- both produce
	// identical (NULL) `local_type`/`local_target` generated-column
	// values, per §4.1's storage-asymmetry note, which applies here too.
	Local commonaddress.Address `json:"local" db:"local,json"`

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
