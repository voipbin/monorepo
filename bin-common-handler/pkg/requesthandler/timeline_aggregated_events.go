package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	tmevent "monorepo/bin-timeline-manager/models/event"
)

// TimelineV1AggregatedEventList sends a request to timeline-manager
// to list aggregated events for a given activeflow.
func (r *requestHandler) TimelineV1AggregatedEventList(ctx context.Context, req *tmevent.AggregatedEventListRequest) (*tmevent.AggregatedEventListResponse, error) {
	uri := "/v1/aggregated-events"

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodPost, "timeline/aggregated-events", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmevent.AggregatedEventListResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
