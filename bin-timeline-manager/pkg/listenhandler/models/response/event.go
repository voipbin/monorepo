package response

import (
	"monorepo/bin-timeline-manager/models/event"
)

// V1DataEventsPost represents the response for event list queries.
type V1DataEventsPost struct {
	Result        []*event.Event `json:"result"`
	NextPageToken string         `json:"next_page_token,omitempty"`
}
