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
	AgentID    uuid.UUID `json:"agent_id"`
}

// processEventWebhookManagerWebhookPublished handles the webhook-manager's webhook_published event.
func (h *subscribeHandler) processEventWebhookManagerWebhookPublished(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventWebhookManagerWebhookPublished",
		"event": m,
	})
	log.Debugf("Received event. event: %s", m.Type)

	wh := &wmwebhook.Webhook{}
	if err := json.Unmarshal(m.Data, wh); err != nil {
		log.Errorf("Could not unmarshal the webhook. err: %v", err)
		return err
	}
	log = log.WithField("customer_id", wh.CustomerID)

	// parse the webhook.data
	tmpWHData, err := json.Marshal(wh.Data)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return err
	}
	whData := &wmwebhook.Data{}
	if err := json.Unmarshal(tmpWHData, whData); err != nil {
		log.Errorf("Could not unmarshal the webhook. err: %v", err)
		return err
	}

	// parse the webhook.data.data
	d := &commonWebhookData{}
	if err := json.Unmarshal(whData.Data, d); err != nil {
		log.Errorf("Could not unmarshal the webhook data. err: %v", err)
		return err
	}

	// create the data
	data, err := json.Marshal(wh.Data)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return err
	}
	log.Debugf("Created data. data: %s", string(data))

	topics, err := h.createTopics(whData.Type, d)
	if err != nil {
		log.Errorf("Could not create the topics")
		return fmt.Errorf("could not create the topics")
	}
	log.Debugf("Created topics. topics: %s", topics)

	for _, topic := range topics {
		if errPub := h.zmqpubHandler.Publish(topic, string(data)); errPub != nil {
			log.Errorf("Could not publish the webhook. err: %v", errPub)
			return errPub
		}
	}

	return nil
}

// createTopic generates the topics
func (h *subscribeHandler) createTopics(messageType string, d *commonWebhookData) ([]string, error) {

	res := []string{}

	tmps := strings.Split(messageType, "_")
	if len(tmps) < 1 {
		return res, fmt.Errorf("wrong type of webhook message. message_type: %s", messageType)
	}

	resource := tmps[0]
	res = append(res, fmt.Sprintf("customer_id:%s:%s:%s", d.CustomerID, resource, d.ID))

	if d.AgentID != uuid.Nil {
		res = append(res, fmt.Sprintf("agent_id:%s:%s:%s", d.AgentID, resource, d.ID))
	}

	return res, nil
}
