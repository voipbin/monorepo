package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/listenhandler/models/request"
)

func (r *requestHandler) WMWebhookPOST(webhookMethod, webhookURI, dataType string, data []byte) error {

	uri := fmt.Sprintf("/v1/webhooks")

	m, err := json.Marshal(request.V1DataWebhooksPost{
		Webhook: webhook.Webhook{
			Method:     webhook.MethodType(webhookMethod),
			WebhookURI: webhookURI,
			DataType:   webhook.DataType(dataType),
			Data:       data,
		},
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestTTS(uri, rabbitmqhandler.RequestMethodPost, resourceTTSSpeeches, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("could not find action")
	}

	return nil
}
