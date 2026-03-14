package response

import (
	"monorepo/bin-timeline-manager/models/event"
)

// V1DataAggregatedEventsPost represents the response for aggregated event list queries.
type V1DataAggregatedEventsPost struct {
	Result        []*event.Event `json:"result"`
	NextPageToken string         `json:"next_page_token,omitempty"`
}
