package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/listenhandler/models/request"
)

// v1DialroutesGet handles /v1/dialroutes GET request
func (h *listenHandler) v1DialroutesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "v1DialroutesGet",
			"request": m,
		},
	)

	// Parse filters from request body
	var reqData request.V1DataDialroutesGet
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &reqData); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, err
		}
	}

	// Extract customer_id and target from filters.
	// Task 6 is formally responsible for fully rewiring this to use the Filters
	// map directly; this minimal extraction keeps the build green.
	//
	// NOTE: json.Unmarshal decodes untyped map values as string (never uuid.UUID),
	// so in practice the string branch below is the only path that fires at runtime.
	var customerID uuid.UUID
	if v, ok := reqData.Filters[route.FieldCustomerID]; ok {
		if s, ok := v.(string); ok {
			customerID = uuid.FromStringOrNil(s)
		}
	}

	var target string
	if v, ok := reqData.Filters[route.FieldTarget]; ok {
		if s, ok := v.(string); ok {
			target = s
		}
	}

	log.WithFields(logrus.Fields{
		"customer_id":         customerID,
		"target":              target,
		"target_provider_ids": reqData.TargetProviderIDs,
		"filters_raw_data":    string(m.Data),
	}).Debug("v1DialroutesGet: Parsed filters from request body")

	tmp, err := h.routeHandler.DialrouteList(ctx, customerID, target, reqData.TargetProviderIDs)
	if err != nil {
		log.Errorf("Could not get routes for dial. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
