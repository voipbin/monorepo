package subscribehandler

import (
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	amresource "monorepo/bin-agent-manager/models/resource"

	"github.com/sirupsen/logrus"
)

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

	// create the data
	data, err := json.Marshal(r.Data)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return err
	}
	log.Debugf("Created data. data: %s", string(data))

	topic := h.createAgentTopic(r)
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
