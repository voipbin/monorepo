package notifyhandler

import (
	"github.com/sirupsen/logrus"
)

func (h *notifyHandler) NotifyEvent(t EventType, webhookURI string, e WebhookMessage) {
	log := logrus.WithFields(
		logrus.Fields{
			"evnet_type":  t,
			"event":       e,
			"webhook_uri": webhookURI,
		},
	)
	log.Debugf("Sending a notify event. event_type: %s", t)

	go h.publishEvent(t, e)
	go h.publishWebhook(t, webhookURI, e)
}
