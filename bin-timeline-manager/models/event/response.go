package event

// EventListResponse represents the response for event list queries.
// Used by the request handler to receive results from timeline-manager.
type EventListResponse struct {
	Result        []*Event `json:"result"`
	NextPageToken string   `json:"next_page_token,omitempty"`
}

// AggregatedEventListResponse represents the response for aggregated event list queries.
type AggregatedEventListResponse struct {
	Result        []*Event `json:"result"`
	NextPageToken string   `json:"next_page_token,omitempty"`
}
