package notifyhandler

import (
	"context"

	"github.com/sirupsen/logrus"
)

func (h *notifyHandler) NotifyEvent(ctx context.Context, eventType EventType, webhookURI string, message WebhookMessage) {
	log := logrus.WithFields(
		logrus.Fields{
			"evnet_type":  eventType,
			"event":       message,
			"webhook_uri": webhookURI,
		},
	)
	log.Debugf("Sending a notify event. event_type: %s", eventType)

	go h.PublishEvent(ctx, eventType, message)
	go h.publishWebhook(ctx, eventType, webhookURI, message)
}
