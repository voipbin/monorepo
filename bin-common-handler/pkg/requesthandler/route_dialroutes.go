package requesthandler

import (
	"context"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
)

// RouteV1DialrouteGets sends a request to route-manager
// to getting a list of dialroute info.
// it returns detail list of dialroute info if it succeed.
func (r *requestHandler) RouteV1DialrouteGets(ctx context.Context, customerID uuid.UUID, target string) ([]rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/dialroutes?customer_id=%s&target=%s", customerID, url.QueryEscape(target))

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodGet, "route/dialroutes", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []rmroute.Route
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
