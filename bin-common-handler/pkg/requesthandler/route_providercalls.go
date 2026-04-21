package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	rmprovidercall "monorepo/bin-route-manager/models/providercall"
	rmrequest "monorepo/bin-route-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// RouteV1ProviderCallCreate sends a request to route-manager to persist a new
// providercall record — capturing the admin's original request info plus the
// IDs of the calls/groupcalls that were already created by the caller via
// CallV1CallsCreate. Returns the persisted record on success.
func (r *requestHandler) RouteV1ProviderCallCreate(
	ctx context.Context,
	customerID uuid.UUID,
	providerID uuid.UUID,
	flowID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	anonymous string,
	callIDs []uuid.UUID,
	groupcallIDs []uuid.UUID,
) (*rmprovidercall.ProviderCall, error) {
	uri := "/v1/providercalls"

	data := &rmrequest.V1DataProviderCallsPost{
		CustomerID:   customerID,
		ProviderID:   providerID,
		FlowID:       flowID,
		Source:       source,
		Destinations: destinations,
		Anonymous:    anonymous,
		CallIDs:      callIDs,
		GroupcallIDs: groupcallIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodPost, "route/providercalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmprovidercall.ProviderCall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1ProviderCallGet sends a request to route-manager to fetch a single
// providercall record by id.
func (r *requestHandler) RouteV1ProviderCallGet(ctx context.Context, providerCallID uuid.UUID) (*rmprovidercall.ProviderCall, error) {
	uri := fmt.Sprintf("/v1/providercalls/%s", providerCallID)

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodGet, "route/providercalls/<providercall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmprovidercall.ProviderCall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RouteV1ProviderCallGets sends a request to route-manager to list
// providercall records with pagination + optional filters (customer_id,
// provider_id). Soft-deleted records are excluded server-side.
func (r *requestHandler) RouteV1ProviderCallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmprovidercall.Field]any) ([]rmprovidercall.ProviderCall, error) {
	q := url.Values{}
	q.Set("page_token", pageToken)
	q.Set("page_size", fmt.Sprintf("%d", pageSize))

	// forward scalar filters as query params. route-manager's listen handler
	// understands customer_id and provider_id today; additional keys are
	// ignored by the handler, so forwarding them is safe but pointless.
	for k, v := range filters {
		q.Set(string(k), fmt.Sprintf("%v", v))
	}

	uri := fmt.Sprintf("/v1/providercalls?%s", q.Encode())

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodGet, "route/providercalls", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []rmprovidercall.ProviderCall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// RouteV1ProviderCallDelete sends a request to route-manager to soft-delete a
// providercall record. Returns the deleted record.
func (r *requestHandler) RouteV1ProviderCallDelete(ctx context.Context, providerCallID uuid.UUID) (*rmprovidercall.ProviderCall, error) {
	uri := fmt.Sprintf("/v1/providercalls/%s", providerCallID)

	tmp, err := r.sendRequestRoute(ctx, uri, sock.RequestMethodDelete, "route/providercalls/<providercall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmprovidercall.ProviderCall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
