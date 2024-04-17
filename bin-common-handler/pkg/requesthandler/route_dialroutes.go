package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// RouteV1DialrouteGets sends a request to route-manager
// to getting a list of dialroute info.
// it returns detail list of dialroute info if it succeed.
func (r *requestHandler) RouteV1DialrouteGets(ctx context.Context, customerID uuid.UUID, target string) ([]rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/dialroutes?customer_id=%s&target=%s", customerID, url.QueryEscape(target))

	res, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceRouteRoutes, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var f []rmroute.Route
	if err := json.Unmarshal([]byte(res.Data), &f); err != nil {
		return nil, err
	}

	return f, nil
}
