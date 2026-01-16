package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-route-manager/pkg/listenhandler/models/request"

	"github.com/sirupsen/logrus"
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

	log.WithFields(logrus.Fields{
		"customer_id":      reqData.CustomerID,
		"target":           reqData.Target,
		"filters_raw_data": string(m.Data),
	}).Debug("v1DialroutesGet: Parsed filters from request body")

	tmp, err := h.routeHandler.DialrouteList(ctx, reqData.CustomerID, reqData.Target)
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
