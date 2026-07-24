package peerevent

import (
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

// PeerEventListRequest represents the request for listing peer_events.
// Used by the request handler to communicate with timeline-manager.
//
// PeerAddresses takes full commonaddress.Address values (not flat
// type/target strings) -- callers (bin-api-manager's resolvePeerPairs)
// already hold Address values (e.g. Contact.Addresses), so this avoids an
// unnecessary Address -> (type, target) -> Address round-trip and keeps
// the request/response filter shape symmetric with contact_interactions'
// own Peer/Local Address-based contract.
type PeerEventListRequest struct {
	CustomerID    uuid.UUID               `json:"customer_id"`
	PeerAddresses []commonaddress.Address `json:"peer_addresses"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}
