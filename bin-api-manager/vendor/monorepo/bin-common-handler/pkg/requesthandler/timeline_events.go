package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	tmevent "monorepo/bin-timeline-manager/models/event"
)

// TimelineV1EventList sends a request to timeline-manager
// to list events matching the given criteria.
func (r *requestHandler) TimelineV1EventList(ctx context.Context, req *tmevent.EventListRequest) (*tmevent.EventListResponse, error) {
	uri := "/v1/events"

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodPost, "timeline/events", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmevent.EventListResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
