package listenhandler

import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/listenhandler/models/request"
)

// processV1WebhooksPost handles POST /v1/webhooks request
func (h *listenHandler) processV1WebhooksPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var reqData request.V1DataWebhooksPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	d, err := json.Marshal(reqData.Data)
	if err != nil {
		logrus.Errorf("Could not marshal the message. message: %v, err: %v", reqData.Data, err)
		return simpleResponse(400), nil
	}

	wh := &webhook.Webhook{
		Method:     reqData.Method,
		WebhookURI: reqData.WebhookURI,
		DataType:   reqData.DataType,
		Data:       d,
	}

	if err := h.whHandler.SendWebhook(wh); err != nil {
		logrus.Debugf("Could not send the webhook correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
