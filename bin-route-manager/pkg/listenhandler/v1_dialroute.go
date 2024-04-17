package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// v1DialroutesGet handles /v1/dialroutes GET request
func (h *listenHandler) v1DialroutesGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1DialroutesGet",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// get customer_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))
	target := u.Query().Get("target")

	tmp, err := h.routeHandler.DialrouteGets(ctx, customerID, target)
	if err != nil {
		log.Errorf("Could not get routes for dial. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
