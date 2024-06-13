package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	whwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventWMWebhookPublished handles the webhook-manager's webhook_published event.
func (h *subscribeHandler) processEventWMWebhookPublished(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventWMWebhookPublished",
		"event": m,
	})

	wh := &whwebhook.Webhook{}
	if err := json.Unmarshal([]byte(m.Data), &wh); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.agentHandler.EventWebhookPublished(ctx, wh); errEvent != nil {
		log.Errorf("Could not handle the webhook published event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the webhook published event.")
	}

	return nil
}
