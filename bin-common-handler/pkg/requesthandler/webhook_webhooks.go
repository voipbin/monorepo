package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	wmwebhook "gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	wmrequest "gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// WebhookV1WebhookSend sends the webhook.
func (r *requestHandler) WebhookV1WebhookSend(ctx context.Context, customerID uuid.UUID, dataType wmwebhook.DataType, messageType string, messageData []byte) error {

	uri := "/v1/webhooks"

	m, err := json.Marshal(wmrequest.V1DataWebhooksPost{
		CustomerID: customerID,
		DataType:   dataType,
		Data: wmwebhook.Data{
			Type: messageType,
			Data: messageData,
		},
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestWebhook(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceWebhookWebhooks, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("could not send an webhook. status: %d", res.StatusCode)
	}

	return nil
}

// WebhookV1WebhookSendToDestination sends the webhook to the given destination.
func (r *requestHandler) WebhookV1WebhookSendToDestination(ctx context.Context, customerID uuid.UUID, destination string, method wmwebhook.MethodType, dataType wmwebhook.DataType, messageData []byte) error {

	uri := "/v1/webhook_destinations"

	m, err := json.Marshal(wmrequest.V1DataWebhookDestinationsPost{
		CustomerID: customerID,
		URI:        destination,
		Method:     method,
		DataType:   dataType,
		Data:       messageData,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestWebhook(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceWebhookWebhooks, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("could not send the webhook. status: %d", res.StatusCode)
	}

	return nil
}
