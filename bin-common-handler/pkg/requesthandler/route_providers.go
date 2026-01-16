package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
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

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodPost, "route/providers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmprovider.Provider
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1ProviderGet sends a request to route-manager
// to getting a detail provider info.
// it returns detail provider info if it succeed.
func (r *requestHandler) RouteV1ProviderGet(ctx context.Context, providerID uuid.UUID) (*rmprovider.Provider, error) {
	uri := fmt.Sprintf("/v1/providers/%s", providerID)

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodGet, "route/providers/<provider-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmprovider.Provider
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1ProviderDelete sends a request to route-manager
// to deleting the provider.
func (r *requestHandler) RouteV1ProviderDelete(ctx context.Context, providerID uuid.UUID) (*rmprovider.Provider, error) {
	uri := fmt.Sprintf("/v1/providers/%s", providerID)

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodDelete, "route/providers/<provider-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmprovider.Provider
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodPut, "route/providers/<provider-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmprovider.Provider
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1ProviderGets sends a request to route-manager
// to getting a list of provider info.
// it returns detail list of provider info if it succeed.
func (r *requestHandler) RouteV1ProviderList(ctx context.Context, pageToken string, pageSize uint64) ([]rmprovider.Provider, error) {
	uri := fmt.Sprintf("/v1/providers?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodGet, "route/providers", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []rmprovider.Provider
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
