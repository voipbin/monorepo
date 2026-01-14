package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/pkg/errors"
)

// RouteV1DialrouteGets sends a request to route-manager
// to getting a list of dialroute info.
// it returns detail list of dialroute info if it succeed.
func (r *requestHandler) RouteV1DialrouteGets(ctx context.Context, filters map[rmroute.Field]any) ([]rmroute.Route, error) {
	uri := "/v1/dialroutes"

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodGet, "route/dialroutes", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []rmroute.Route
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
