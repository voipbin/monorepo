package webhookhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
)

// SendWebhook sends the webhook to the given uri with the given method and data.
func (h *webhookHandler) SendWebhook(w *webhook.Webhook) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"webhook": w,
		},
	)
	log.Debugf("Sending an webhook. customer_id: %s", w.CustomerID)

	m, err := h.messageTargetHandler.Get(ctx, w.CustomerID)
	if err != nil {
		log.Errorf("Could not get message target. err: %v", err)
		return fmt.Errorf("could not get message target. err: %v", err)
	}

	if m.WebhookURI == "" {
		log.Infof("Invalid uri target. uri: %s", m.WebhookURI)
		return fmt.Errorf("invalid uri target. uri: %s", m.WebhookURI)
	}

	// send message
	go func() {
		res, err := h.sendMessage(m.WebhookURI, string(m.WebhookMethod), string(w.DataType), []byte(w.Data))
		if err != nil {
			log.Errorf("Could not send a request. err: %v", err)
			return
		}
		log.Debugf("Sent the request correctly. method: %s, uri: %s, res: %d", m.WebhookMethod, m.WebhookURI, res.StatusCode)

	}()

	return nil
}
