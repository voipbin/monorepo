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
type PeerEventRow struct {
	Timestamp   time.Time
	CustomerID  uuid.UUID
	Publisher   string
	EventType   string
	ReferenceID uuid.UUID
	Direction   string

	PeerType   string
	PeerTarget string

	LocalType   string
	LocalTarget string

	Data string
}
