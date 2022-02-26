package webhookhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
)

// SendWebhook sends the webhook to the given uri with the given method and data.
// func (h *webhookHandler) SendWebhook(w *webhook.Webhook) error {
func (h *webhookHandler) SendWebhook(ctx context.Context, customerID uuid.UUID, dataType webhook.DataType, data json.RawMessage) error {
	log := logrus.WithFields(
		logrus.Fields{
			"customer_id": customerID,
		},
	)
	log.WithFields(logrus.Fields{
		"data_type": dataType,
		"data":      data,
	}).Debugf("Sending an webhook. customer_id: %s", customerID)

	m, err := h.messageTargetHandler.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get message target. err: %v", err)
		return fmt.Errorf("could not get message target. err: %v", err)
	}

	if m.WebhookURI == "" {
		// no place to send
		log.Infof("Invalid uri target. uri: %s", m.WebhookURI)
		return nil
	}

	// send message
	go func() {
		res, err := h.sendMessage(m.WebhookURI, string(m.WebhookMethod), string(dataType), data)
		if err != nil {
			log.Errorf("Could not send a request. err: %v", err)
			return
		}
		log.Debugf("Sent the request correctly. method: %s, uri: %s, res: %d", m.WebhookMethod, m.WebhookURI, res.StatusCode)

	}()

	return nil
}
