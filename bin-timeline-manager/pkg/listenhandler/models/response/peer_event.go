package response

import (
	"monorepo/bin-timeline-manager/models/peerevent"
)

// V1DataPeerEventsGet represents the response for peer_events list queries.
type V1DataPeerEventsGet struct {
	Result        []*peerevent.PeerEvent `json:"result"`
	NextPageToken string                 `json:"next_page_token,omitempty"`
}
