package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	wmwebhook "gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
)

type commonWebhookData struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
}

// processEventWebhookManagerWebhookPublished handles the webhook-manager's webhook_published event.
func (h *subscribeHandler) processEventWebhookManagerWebhookPublished(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventWebhookManagerWebhookPublished",
			"event": m,
		},
	)
	log.Debugf("Received event. event: %s", m.Type)

	e := &wmwebhook.Data{}
	if err := json.Unmarshal(m.Data, e); err != nil {
		log.Errorf("Could not unmarshal the webhook. err: %v", err)
		return err
	}
	log.Debugf("Parsed webhook data. type: %s", e.Type)

	d := &commonWebhookData{}
	if err := json.Unmarshal(e.Data, d); err != nil {
		log.Errorf("Could not unmarshal the webhook data. err: %v", err)
		return err
	}
	log.Debugf("Received webhook. type: %s, id: %s", e.Type, d.ID)

	// parse resource
	tmps := strings.Split(e.Type, "_")
	if len(tmps) < 1 {
		log.Errorf("Wrong type of webhook message. message: %v", e)
		return fmt.Errorf("wrong type of webhook message. message: %v", e)
	}
	resource := tmps[0]
	topic := fmt.Sprintf("%s:%s:%s", d.CustomerID, resource, d.ID)
	log.Debugf("Publishing the data. resource: %s, topic: %s", resource, topic)

	if errPub := h.zmqpubHandler.Publish(topic, string(m.Data)); errPub != nil {
		log.Errorf("Could not publish the webhook. err: %v", errPub)
		return errPub
	}

	return nil
}
