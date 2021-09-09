package notifyhandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func (h *notifyHandler) publishWebhook(t EventType, c WebhookMessage) {
	log := logrus.WithFields(
		logrus.Fields{
			"call":       c,
			"evnet_type": t,
		},
	)
	log.Debugf("Sending webhook event. event_type: %s, message: %s", t, c)

	webhookURI := c.GetWebhookURI()
	if webhookURI == "" {
		// no webhook uri
		return
	}

	// create webhook event
	m, err := c.CreateWebhookEvent(string(t))
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := h.reqHandler.WMWebhookPOST("POST", webhookURI, dataTypeJSON, string(t), m); err != nil {
		log.Errorf("Could not publish the webhook. err: %v", err)
		return
	}

	return
}

func (h *notifyHandler) publishEvent(t EventType, c interface{}) {
	// create event
	m, err := json.Marshal(c)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := h.publishNotify(string(t), dataTypeJSON, m, requestTimeoutDefault); err != nil {
		log.Errorf("Could not publish the call event. err: %v", err)
		return
	}
	return
}
