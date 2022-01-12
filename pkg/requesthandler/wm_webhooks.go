package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/listenhandler/models/request"
)

// WMV1WebhookSend sends the webhook.
func (r *requestHandler) WMV1WebhookSend(ctx context.Context, webhookMethod, webhookURI, dataType, messageType string, messageData []byte) error {

	uri := "/v1/webhooks"

	m, err := json.Marshal(request.V1DataWebhooksPost{
		Method:     webhook.MethodType(webhookMethod),
		WebhookURI: webhookURI,
		DataType:   webhook.DataType(dataType),
		Data: request.WebhookData{
			Type: messageType,
			Data: messageData,
		},
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestWM(uri, rabbitmqhandler.RequestMethodPost, resourceTTSSpeeches, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("could not find action")
	}

	return nil
}
