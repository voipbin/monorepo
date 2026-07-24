package peerevent

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
)

// PeerEvent represents a single peer_events row from ClickHouse.
// Publisher here is a SYNTHETIC derived label ("call" / "conversation_message" /
// "conversation"), NOT the raw wire event.Publisher value (which is only ever
// "call-manager" or "conversation-manager"). See buildPeerEventRows in
// bin-timeline-manager/pkg/subscribehandler for the derivation on the write side.
type PeerEvent struct {
	Timestamp   time.Time       `json:"timestamp"`
	CustomerID  uuid.UUID       `json:"customer_id"`
	Publisher   string          `json:"publisher"`
	EventType   string          `json:"event_type"`
	ReferenceID uuid.UUID       `json:"reference_id"`
	Direction   string          `json:"direction"`
	PeerType    string          `json:"peer_type"`
	PeerTarget  string          `json:"peer_target"`
	LocalType   string          `json:"local_type"`
	LocalTarget string          `json:"local_target"`
	Data        json.RawMessage `json:"data"`
}
