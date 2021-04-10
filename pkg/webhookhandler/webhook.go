package webhookhandler

import (
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
)

// SendWebhook sends the webhook to the given uri with the given method and data.
func (h *webhookHandler) SendWebhook(wh *webhook.Webhook) error {

	log := logrus.WithFields(
		logrus.Fields{
			"webhook": wh,
		},
	)
	log.Debugf("Sending an webhook. method: %s, uri: %s", wh.Method, wh.WebhookURI)

	res, err := h.SendMessage(wh.WebhookURI, string(wh.Method), string(wh.DataType), wh.Data)
	if err != nil {
		// should we re-post this request?
		log.Errorf("Could not send a request. err: %v", err)
		return err
	}

	log.Debugf("Sent the request correctly. method: %s, uri: %s, res: %d", wh.Method, wh.WebhookURI, res.StatusCode)
	return nil
}
