package subscribehandler

import (
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	amresource "monorepo/bin-agent-manager/models/resource"

	"github.com/sirupsen/logrus"
)

type resourceWebhookData struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// processEventAgentManagerResourcePublished handles the webhook-manager's webhook_published event.
func (h *subscribeHandler) processEventAgentManagerResourcePublished(m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventAgentManagerResourcePublished",
		"event": m,
	})
	log.Debugf("Received agent-manager event. event: %s", m.Type)

	r := &amresource.Resource{}
	if err := json.Unmarshal(m.Data, r); err != nil {
		log.Errorf("Could not unmarshal the webhook. err: %v", err)
		return err
	}
	log = log.WithField("customer_id", r.CustomerID)

	topic := h.createAgentTopic(r)
	log.Debugf("Created agent topic. topic: %s", topic)

	// create agent resource
	tmp := resourceWebhookData{
		Type: m.Type,
		Data: m.Data,
	}

	// create the data
	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return err
	}
	log.Debugf("Created data. data: %s", string(data))

	if errPub := h.zmqpubHandler.Publish(topic, string(data)); errPub != nil {
		log.Errorf("Could not publish the webhook. err: %v", errPub)
		return errPub
	}

	return nil
}

// createTopic generates the topics
func (h *subscribeHandler) createAgentTopic(r *amresource.Resource) string {

	res := fmt.Sprintf("agent_id:%s:%s:%s", r.OwnerID, r.ReferenceType, r.ReferenceID)
	return res
}
