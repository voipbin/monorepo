package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

// TimelineV1PeerEventList sends a request to timeline-manager to list
// peer_events rows matching the given (peer_type, peer_target) pairs.
// GET, not POST — mirrors TimelineV1AnalysisList's shape:
// customer_id/page_token/page_size in the query string, the peer_pairs
// array filter JSON-marshaled into the body.
func (r *requestHandler) TimelineV1PeerEventList(ctx context.Context, req *tmpeerevent.PeerEventListRequest) (*tmpeerevent.PeerEventListResponse, error) {
	uri := fmt.Sprintf(
		"/v1/peer-events?customer_id=%s&page_token=%s&page_size=%d",
		req.CustomerID.String(), url.QueryEscape(req.PageToken), req.PageSize,
	)

	m, err := json.Marshal(struct {
		PeerPairs []tmpeerevent.PeerPair `json:"peer_pairs"`
	}{PeerPairs: req.PeerPairs})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodGet, "timeline/peer-events", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmpeerevent.PeerEventListResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
