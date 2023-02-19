package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/listenhandler/models/request"
)

// processV1ConferencecallsIDGet handles /v1/conferencecalls/<id> GET request
func (h *listenHandler) processV1ConferencecallsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencecallsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cc, err := h.conferencecallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not remove the call from the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cc)
	if err != nil {
		log.Errorf("Could not marshal the conferencecall. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencecallsIDDelete handles /v1/conferencecalls/<id> DELETE request
func (h *listenHandler) processV1ConferencecallsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencecallsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cc, err := h.conferencecallHandler.Terminate(ctx, id)
	if err != nil {
		log.Errorf("Could not remove the call from the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cc)
	if err != nil {
		log.Errorf("Could not marshal the conferencecall. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencecallsIDHealthCheckPost handles /v1/conferencecalls/<id>/health-check POST request
func (h *listenHandler) processV1ConferencecallsIDHealthCheckPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencecallsIDHealthCheckPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataConferencecallsIDHealthCheckPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// health check run in a go routine
	go h.conferencecallHandler.HealthCheck(ctx, id, data.RetryCount)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}
