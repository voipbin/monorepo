package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	rmroute "monorepo/bin-route-manager/models/route"
	rmrequest "monorepo/bin-route-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodPost, "route/routes", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmroute.Route
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteV1RouteGet sends a request to route-manager
// to getting a detail route info.
// it returns detail route info if it succeed.
func (r *requestHandler) RouteV1RouteGet(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/routes/%s", routeID)

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodGet, "route/routes/<route-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmroute.Route
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteV1RouteDelete sends a request to route-manager
// to deleting the route.
func (r *requestHandler) RouteV1RouteDelete(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/routes/%s", routeID)

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodDelete, "route/routes/<route-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmroute.Route
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
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

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodPut, "route/routes/<route-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmroute.Route
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteV1RouteGetsByCustomerID sends a request to route-manager
// to getting a list of route info.
// it returns detail list of route info if it succeed.
func (r *requestHandler) RouteV1RouteGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/routes?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	res, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodGet, "route/routes", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// RouteV1RouteGets sends a request to route-manager
// to getting a list of route info.
// it returns detail list of route info if it succeed.
func (r *requestHandler) RouteV1RouteGets(ctx context.Context, pageToken string, pageSize uint64) ([]rmroute.Route, error) {
	uri := fmt.Sprintf("/v1/routes?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	res, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodGet, "route/routes", requestTimeoutDefault, 0, ContentTypeNone, nil)
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
