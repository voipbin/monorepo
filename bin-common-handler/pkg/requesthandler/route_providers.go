package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	rmprovider "monorepo/bin-route-manager/models/provider"
	rmrequest "monorepo/bin-route-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// RouteV1ProviderCreate sends a request to route-manager
// to creating a provider.
// it returns created provider if it succeed.
func (r *requestHandler) RouteV1ProviderCreate(
	ctx context.Context,
	provierType rmprovider.Type,
	hostname string,
	techPrefix string,
	techPostfix string,
	techHeaders map[string]string,
	name string,
	detail string,
) (*rmprovider.Provider, error) {
	uri := "/v1/providers"

	data := &rmrequest.V1DataProvidersPost{
		Type:        provierType,
		Hostname:    hostname,
		TechPrefix:  techPrefix,
		TechPostfix: techPostfix,
		TechHeaders: techHeaders,
		Name:        name,
		Detail:      detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceRouteProviders, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmprovider.Provider
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteV1ProviderGet sends a request to route-manager
// to getting a detail provider info.
// it returns detail provider info if it succeed.
func (r *requestHandler) RouteV1ProviderGet(ctx context.Context, providerID uuid.UUID) (*rmprovider.Provider, error) {
	uri := fmt.Sprintf("/v1/providers/%s", providerID)

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceRouteProviders, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmprovider.Provider
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteV1ProviderDelete sends a request to route-manager
// to deleting the provider.
func (r *requestHandler) RouteV1ProviderDelete(ctx context.Context, providerID uuid.UUID) (*rmprovider.Provider, error) {
	uri := fmt.Sprintf("/v1/providers/%s", providerID)

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceRouteProviders, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmprovider.Provider
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteV1ProviderUpdate sends a request to route-manager
// to update the detail provider info.
// it returns updated provider info if it succeed.
func (r *requestHandler) RouteV1ProviderUpdate(
	ctx context.Context,
	providerID uuid.UUID,
	providerType rmprovider.Type,
	hostname string,
	techPrefix string,
	techPostfix string,
	techHeaders map[string]string,
	name string,
	detail string,
) (*rmprovider.Provider, error) {
	uri := fmt.Sprintf("/v1/providers/%s", providerID)

	data := &rmrequest.V1DataProvidersIDPut{
		Type:        providerType,
		Hostname:    hostname,
		TechPrefix:  techPrefix,
		TechPostfix: techPostfix,
		TechHeaders: techHeaders,
		Name:        name,
		Detail:      detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceRouteProviders, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmprovider.Provider
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteV1ProviderGets sends a request to route-manager
// to getting a list of provider info.
// it returns detail list of provider info if it succeed.
func (r *requestHandler) RouteV1ProviderGets(ctx context.Context, pageToken string, pageSize uint64) ([]rmprovider.Provider, error) {
	uri := fmt.Sprintf("/v1/providers?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	tmp, err := r.sendRequestRoute(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceRouteProviders, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []rmprovider.Provider
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}
