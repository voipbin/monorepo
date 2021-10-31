package notifyhandler

import (
	"github.com/sirupsen/logrus"
)

func (h *notifyHandler) NotifyEvent(eventType EventType, webhookURI string, message WebhookMessage) {
	log := logrus.WithFields(
		logrus.Fields{
			"evnet_type":  eventType,
			"event":       message,
			"webhook_uri": webhookURI,
		},
	)
	log.Debugf("Sending a notify event. event_type: %s", eventType)

	go h.publishEvent(eventType, message)
	go h.publishWebhook(eventType, webhookURI, message)
}
