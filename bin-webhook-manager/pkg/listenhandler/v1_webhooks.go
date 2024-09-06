package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"

	"monorepo/bin-webhook-manager/pkg/listenhandler/models/request"
)

// processV1WebhooksPost handles POST /v1/webhooks request
func (h *listenHandler) processV1WebhooksPost(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1WebhooksPost",
		},
	)
	log.WithField("request", m).Debugf("Sending a webhook message.")

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var req request.V1DataWebhooksPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	d, err := json.Marshal(req.Data)
	if err != nil {
		logrus.Errorf("Could not marshal the message. message: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}

	if err := h.whHandler.SendWebhookToCustomer(ctx, req.CustomerID, req.DataType, d); err != nil {
		logrus.Debugf("Could not send the webhook correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1WebhookDestinationsPost handles POST /v1/webhook_destinations request
func (h *listenHandler) processV1WebhookDestinationsPost(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1WebhookDestinationsPost",
		},
	)
	log.WithField("request", m).Debugf("Sending a webhook message.")

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var req request.V1DataWebhookDestinationsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	d, err := json.Marshal(req.Data)
	if err != nil {
		log.Errorf("Could not marshal the message. message: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}

	if err := h.whHandler.SendWebhookToURI(ctx, req.CustomerID, req.URI, req.Method, req.DataType, d); err != nil {
		log.Debugf("Could not send the webhook correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
