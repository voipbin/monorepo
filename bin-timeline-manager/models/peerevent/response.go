package peerevent

// PeerEventListResponse represents the response for peer_events list queries.
// Used by the request handler to receive results from timeline-manager.
type PeerEventListResponse struct {
	Result        []*PeerEvent `json:"result"`
	NextPageToken string       `json:"next_page_token,omitempty"`
}
