package peerevent

import (
	"github.com/gofrs/uuid"
)

// PeerPair is a (peer_type, peer_target) pair used to filter peer_events.
type PeerPair struct {
	PeerType   string `json:"peer_type"`
	PeerTarget string `json:"peer_target"`
}

// PeerEventListRequest represents the request for listing peer_events.
// Used by the request handler to communicate with timeline-manager.
type PeerEventListRequest struct {
	CustomerID uuid.UUID  `json:"customer_id"`
	PeerPairs  []PeerPair `json:"peer_pairs"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}
