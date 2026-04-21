package requesthandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/sock"
	rmrequest "monorepo/bin-route-manager/pkg/listenhandler/models/request"
	rmroute "monorepo/bin-route-manager/models/route"
)

// RouteV1DialrouteList sends a request to route-manager
// to getting a list of dialroute info.
// it returns detail list of dialroute info if it succeed.
func (r *requestHandler) RouteV1DialrouteList(
	ctx context.Context,
	filters map[rmroute.Field]any,
	targetProviderIDs []uuid.UUID,
) ([]rmroute.Route, error) {
	uri := "/v1/dialroutes"

	// Populate legacy top-level CustomerID/Target alongside the Filters map
	// so an older route-manager (deployed during a rolling update) can still
	// parse the request. See V1DataDialroutesGet for the backward-compat contract.
	var legacyCustomerID uuid.UUID
	var legacyTarget string
	if v, ok := filters[rmroute.FieldCustomerID]; ok {
		if id, ok := v.(uuid.UUID); ok {
			legacyCustomerID = id
		}
	}
	if v, ok := filters[rmroute.FieldTarget]; ok {
		if s, ok := v.(string); ok {
			legacyTarget = s
		}
	}

	m, err := json.Marshal(rmrequest.V1DataDialroutesGet{
		CustomerID:        legacyCustomerID,
		Target:            legacyTarget,
		Filters:           filters,
		TargetProviderIDs: targetProviderIDs,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal request data")
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
