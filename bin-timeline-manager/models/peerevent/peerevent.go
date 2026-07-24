package peerevent

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

// PeerEvent represents a single peer_events row from ClickHouse.
// Publisher here is a SYNTHETIC derived label ("call" / "conversation_message" /
// "conversation"), NOT the raw wire event.Publisher value (which is only ever
// "call-manager" or "conversation-manager"). See buildPeerEventRows in
// bin-timeline-manager/pkg/subscribehandler for the derivation on the write side.
//
// Peer/Local are the full commonaddress.Address (Type/Target/TargetName/
// Name/Detail) -- there is deliberately no flat PeerType/PeerTarget field
// here. Those exist only as internal, search-only columns inside
// dbhandler.PeerEventRow (ClickHouse's ORDER BY/WHERE index); they are
// never part of this response-facing shape. Mirrors contact_interactions'
// pattern (interaction.Interaction has Peer/Local commonaddress.Address
// fields, not flat Type/Target fields).
type PeerEvent struct {
	Timestamp   time.Time             `json:"timestamp"`
	CustomerID  uuid.UUID             `json:"customer_id"`
	Publisher   string                `json:"publisher"`
	EventType   string                `json:"event_type"`
	ReferenceID uuid.UUID             `json:"reference_id"`
	Direction   string                `json:"direction"`
	Peer        commonaddress.Address `json:"peer"`
	Local       commonaddress.Address `json:"local"`
	Data        json.RawMessage       `json:"data"`
}
