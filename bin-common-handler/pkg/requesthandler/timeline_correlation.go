package requesthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
	tmcorrelation "monorepo/bin-timeline-manager/models/correlation"
)

// TimelineV1CorrelationGet sends a request to timeline-manager to get
// the correlation graph for a resource id (GET /v1/correlations/<resource_id>).
func (r *requestHandler) TimelineV1CorrelationGet(ctx context.Context, resourceID uuid.UUID) (*tmcorrelation.Correlation, error) {
	uri := fmt.Sprintf("/v1/correlations/%s", resourceID.String())

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodGet, "timeline/correlations", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmcorrelation.Correlation
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
