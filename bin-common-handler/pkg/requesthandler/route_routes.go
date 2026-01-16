package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	rmroute "monorepo/bin-route-manager/models/route"
	rmrequest "monorepo/bin-route-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// RouteV1RouteCreate sends a request to route-manager
// to creating a route.
// it returns created route if it succeed.
func (r *requestHandler) RouteV1RouteCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	providerID uuid.UUID,
	priority int,
	target string,
) (*rmroute.Route, error) {
	uri := "/v1/routes"

	data := &rmrequest.V1DataRoutesPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
		ProviderID: providerID,
		Priority:   priority,
		Target:     target,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodPost, "route/routes", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmroute.Route
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1RouteGet sends a request to route-manager
// to getting a detail route info.
// it returns detail route info if it succeed.
func (r *requestHandler) RouteV1RouteGet(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/routes/%s", routeID)

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodGet, "route/routes/<route-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmroute.Route
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1RouteDelete sends a request to route-manager
// to deleting the route.
func (r *requestHandler) RouteV1RouteDelete(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/routes/%s", routeID)

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodDelete, "route/routes/<route-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmroute.Route
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1RouteUpdate sends a request to route-manager
// to update the detail route info.
// it returns updated route info if it succeed.
func (r *requestHandler) RouteV1RouteUpdate(
	ctx context.Context,
	routeID uuid.UUID,
	name string,
	detail string,
	providerID uuid.UUID,
	priority int,
	target string,
) (*rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/routes/%s", routeID)

	data := &rmrequest.V1DataRoutesIDPut{
		Name:       name,
		Detail:     detail,
		ProviderID: providerID,
		Priority:   priority,
		Target:     target,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodPut, "route/routes/<route-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmroute.Route
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1RouteList sends a request to route-manager
// to getting a list of route info.
// it returns detail list of route info if it succeed.
func (r *requestHandler) RouteV1RouteList(ctx context.Context, pageToken string, pageSize uint64, filters map[rmroute.Field]any) ([]rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/routes?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodGet, "route/routes", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []rmroute.Route
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
