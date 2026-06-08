package requesthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
	tmcorrelation "monorepo/bin-timeline-manager/models/correlation"
)

// TimelineV1ResourceCorrelationGet sends a request to timeline-manager to get
// the correlation graph for a resource id (GET /v1/correlations/<resource_id>).
func (r *requestHandler) TimelineV1ResourceCorrelationGet(ctx context.Context, resourceID uuid.UUID) (*tmcorrelation.ResourceCorrelationResponse, error) {
	uri := fmt.Sprintf("/v1/correlations/%s", resourceID.String())

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodGet, "timeline/correlations", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmcorrelation.ResourceCorrelationResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
