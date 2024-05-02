package webhookhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webhook-manager/models/webhook"
)

// SendWebhookToCustomer sends the webhook to the given customerID with the given method and data.
func (h *webhookHandler) SendWebhookToCustomer(ctx context.Context, customerID uuid.UUID, dataType webhook.DataType, data json.RawMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SendWebhookToCustomer",
		"customer_id": customerID,
	})
	log.WithFields(logrus.Fields{
		"data_type": dataType,
		"data":      data,
	}).Debugf("Sending an webhook. customer_id: %s", customerID)

	m, err := h.accoutHandler.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return fmt.Errorf("could not get account. err: %v", err)
	}

	if m.WebhookURI != "" {
		// send webhook message
		go func() {
			res, err := h.sendMessage(m.WebhookURI, string(m.WebhookMethod), string(dataType), data)
			if err != nil {
				log.Errorf("Could not send a request. err: %v", err)
				return
			}
			log.Debugf("Sent the request correctly. method: %s, uri: %s, res: %d", m.WebhookMethod, m.WebhookURI, res.StatusCode)
		}()
	}

	wh := &webhook.Webhook{
		CustomerID: customerID,
		DataType:   dataType,
		Data:       data,
	}
	h.notifyHandler.PublishEvent(ctx, webhook.EventTypeWebhookPublished, wh)

	return nil
}

// SendWebhookToURI sends the webhook to the given uri with the given method and data.
func (h *webhookHandler) SendWebhookToURI(ctx context.Context, customerID uuid.UUID, uri string, method webhook.MethodType, dataType webhook.DataType, data json.RawMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SendWebhookToURI",
		"customer_id": customerID,
		"uri":         uri,
	})
	log.WithFields(logrus.Fields{
		"data_type": dataType,
		"data":      data,
	}).Debugf("Sending an webhook. customer_id: %s", customerID)

	// send message
	go func() {
		res, err := h.sendMessage(uri, string(method), string(dataType), data)
		if err != nil {
			log.Errorf("Could not send a request. err: %v", err)
			return
		}
		log.Debugf("Sent the request correctly. method: %s, uri: %s, res: %d", method, uri, res.StatusCode)
	}()

	wh := &webhook.Webhook{
		CustomerID: customerID,
		DataType:   dataType,
		Data:       data,
	}
	h.notifyHandler.PublishEvent(ctx, webhook.EventTypeWebhookPublished, wh)

	return nil
}
