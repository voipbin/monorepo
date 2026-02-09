package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

type countByCustomerRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

type countByCustomerResponse struct {
	Count int `json:"count"`
}

// processV1ConferencesCountByCustomerGet handles GET /v1/conferences/count_by_customer
func (h *listenHandler) processV1ConferencesCountByCustomerGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesCountByCustomerGet",
		"request": m,
	})

	var req countByCustomerRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the request data. err: %v", err)
		return simpleResponse(400), nil
	}

	count, err := h.conferenceHandler.CountByCustomerID(ctx, req.CustomerID)
	if err != nil {
		log.Errorf("Could not get conference count. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(&countByCustomerResponse{Count: count})
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
