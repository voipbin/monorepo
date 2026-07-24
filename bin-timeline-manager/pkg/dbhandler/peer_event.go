package dbhandler

import (
	"time"

	"github.com/gofrs/uuid"
)

// PeerEventRow represents a single peer_events row for batch insert.
// Publisher here is a SYNTHETIC derived label ("call" / "conversation_message" /
// "conversation"), NOT the raw wire event.Publisher value (which is only ever
// "call-manager" or "conversation-manager"). See buildPeerEventRows in
// pkg/subscribehandler for the derivation.
//
// Peer/Local carry the full commonaddress.Address (JSON-serialized) -- the
// external, response-facing shape. PeerType/PeerTarget/LocalType/LocalTarget
// are flat, INTERNAL-ONLY columns populated from the same Address values at
// insert time (mirrors contact_interactions' JSON+generated-column split,
// but computed in Go rather than by the database, since ClickHouse has no
// STORED GENERATED COLUMN equivalent). They exist purely so peer_events'
// ORDER BY (customer_id, peer_type, peer_target, timestamp) index and
// WHERE-clause search stay on physical columns -- never exposed by the
// read API.
type PeerEventRow struct {
	Timestamp   time.Time
	CustomerID  uuid.UUID
	Publisher   string
	EventType   string
	ReferenceID uuid.UUID
	Direction   string

	Peer  string // JSON(commonaddress.Address), response-facing
	Local string // JSON(commonaddress.Address), response-facing

	PeerType   string // internal-only: ORDER BY / WHERE search
	PeerTarget string // internal-only: ORDER BY / WHERE search

	LocalType   string // internal-only: WHERE search
	LocalTarget string // internal-only: WHERE search

	Data string
}
