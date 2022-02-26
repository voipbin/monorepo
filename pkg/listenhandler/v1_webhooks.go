package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/listenhandler/models/request"
)

// processV1WebhooksPost handles POST /v1/webhooks request
func (h *listenHandler) processV1WebhooksPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()
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

	if err := h.whHandler.SendWebhook(ctx, req.CustomerID, req.DataType, d); err != nil {
		logrus.Debugf("Could not send the webhook correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
