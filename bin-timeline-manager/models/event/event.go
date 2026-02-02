package event

import (
	"encoding/json"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Event represents a single event from ClickHouse.
type Event struct {
	Timestamp string                    `json:"timestamp"`
	EventType string                    `json:"event_type"`
	Publisher commonoutline.ServiceName `json:"publisher"`
	DataType  string                    `json:"data_type"`
	Data      json.RawMessage           `json:"data"`
}

// EventListResponse represents the response for event list queries.
type EventListResponse struct {
	Result        []*Event `json:"result"`
	NextPageToken string   `json:"next_page_token,omitempty"`
}
