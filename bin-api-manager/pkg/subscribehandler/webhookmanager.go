package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

type commonWebhookData struct {
	commonidentity.Identity
	commonidentity.Owner
	AIcallID uuid.UUID `json:"aicall_id,omitempty"`
	ChatID   uuid.UUID `json:"chat_id,omitempty"`
}

// getServiceNamespace maps RabbitMQ publisher name to topic namespace
func (h *subscribeHandler) getServiceNamespace(publisher string) string {
	namespaces := map[string]string{
		"talk-manager":         "talk",
		"message-manager":      "message",
		"call-manager":         "call",
		"conference-manager":   "conference",
		"flow-manager":         "flow",
		"agent-manager":        "agent",
		"billing-manager":      "billing",
		"campaign-manager":     "campaign",
		"conversation-manager": "conversation",
		"webhook-manager":      "webhook",
	}

	if ns, ok := namespaces[publisher]; ok {
		return ns
	}

	// Default: use publisher name as-is
	return publisher
}

// processEventWebhookManagerWebhookPublished handles the webhook-manager's webhook_published event.
func (h *subscribeHandler) processEventWebhookManagerWebhookPublished(ctx context.Context, m *sock.Event) error {
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

	topics, err := h.createTopics(ctx, whData.Type, d, m.Publisher)
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

// processEventWebhookManagerRoutingKeyedEvent handles events arriving via the NEW topic
// exchange (VOIP-1258 §6/§8), published by bin-webhook-manager's publishRoutingKeyedEvent. Unlike
// processEventWebhookManagerWebhookPublished above, m.Data here is ALREADY the unwrapped resource
// object (bin-webhook-manager did the envelope unwrap at publish time), and m.Type is the REAL
// resource event type (e.g. "call_created"), not the fixed "webhook_published" constant. This
// function mirrors createTopics()'s topic-string generation so ZMQ SUB clients (which subscribe
// using the OLD-FORMAT prefix, e.g. "customer_id:<id>:call") continue to receive these events
// exactly as they did via the old fanout path -- see this file's createTopics for the format.
func (h *subscribeHandler) processEventWebhookManagerRoutingKeyedEvent(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventWebhookManagerRoutingKeyedEvent",
		"event": m,
	})
	log.Debugf("Received routing-keyed event. type: %s", m.Type)

	d := &commonWebhookData{}
	if err := json.Unmarshal(m.Data, d); err != nil {
		log.Errorf("Could not unmarshal the resource data. err: %v", err)
		return err
	}

	// resource is the same first-underscore-segment used by bin-webhook-manager's
	// publishRoutingKeyedEvent to build the routing key (e.g. "call" from "call_created"), and
	// happens to already match this file's getServiceNamespace() output for the corresponding
	// publisher (e.g. "call-manager" -> "call") -- use it directly as the service namespace
	// instead of trying to recover the original publisher name, which is not preserved on this
	// path (m.Publisher is always "webhook-manager" here, since that's who republished it).
	tmps := strings.SplitN(m.Type, "_", 2)
	if len(tmps) < 2 {
		log.Errorf("Wrong event type format. event_type: %s", m.Type)
		return fmt.Errorf("wrong event type format. event_type: %s", m.Type)
	}
	resource := tmps[0]

	topics, err := h.createTopics(ctx, m.Type, d, resource)
	if err != nil {
		log.Errorf("Could not create the topics. err: %v", err)
		return fmt.Errorf("could not create the topics")
	}
	log.Debugf("Created topics. topics: %s", topics)

	for _, topic := range topics {
		if errPub := h.zmqpubHandler.Publish(topic, string(m.Data)); errPub != nil {
			log.Errorf("Could not publish the event. err: %v", errPub)
			return errPub
		}
	}

	return nil
}

// createTopics generates the topics
func (h *subscribeHandler) createTopics(ctx context.Context, messageType string, d *commonWebhookData, publisher string) ([]string, error) {

	res := []string{}

	tmps := strings.Split(messageType, "_")
	if len(tmps) < 2 {
		return nil, fmt.Errorf("wrong type of webhook message. message_type: %s", messageType)
	}

	// Get service namespace from publisher
	service := h.getServiceNamespace(publisher)

	// OLD FORMAT (backward compatible):
	resource := tmps[0]

	switch resource {
	case "aimessage":
		if d.CustomerID != uuid.Nil {
			res = append(res, fmt.Sprintf("customer_id:%s:aicall:%s", d.CustomerID, d.AIcallID))
		}

	case "chat", "chatmessage", "chatparticipant":
		chatID := d.ChatID
		if chatID == uuid.Nil {
			chatID = d.ID
		}

		if d.CustomerID != uuid.Nil {
			res = append(res, fmt.Sprintf("customer_id:%s:chat:%s", d.CustomerID, chatID))
		}
		if d.OwnerID != uuid.Nil {
			res = append(res, fmt.Sprintf("agent_id:%s:chat:%s", d.OwnerID, chatID))
		}

		// Fan-out to all chat participants
		if chatID != uuid.Nil {
			participants, err := h.reqHandler.TalkV1ParticipantList(ctx, chatID)
			if err == nil {
				for _, p := range participants {
					if p.OwnerID == d.OwnerID {
						continue
					}
					// Old format
					res = append(res, fmt.Sprintf("agent_id:%s:chat:%s", p.OwnerID, chatID))
					// New format
					res = append(res, fmt.Sprintf("agent_id:%s:%s:%s:%s", p.OwnerID, service, messageType, d.ID))
				}
			}
		}

	default:
		if d.CustomerID != uuid.Nil {
			res = append(res, fmt.Sprintf("customer_id:%s:%s:%s", d.CustomerID, resource, d.ID))
		}
		if d.OwnerID != uuid.Nil {
			res = append(res, fmt.Sprintf("agent_id:%s:%s:%s", d.OwnerID, resource, d.ID))
		}
	}

	// NEW FORMAT (service-namespaced):
	if d.CustomerID != uuid.Nil {
		res = append(res, fmt.Sprintf("customer_id:%s:%s:%s:%s", d.CustomerID, service, messageType, d.ID))
	}
	if d.OwnerID != uuid.Nil {
		res = append(res, fmt.Sprintf("agent_id:%s:%s:%s:%s", d.OwnerID, service, messageType, d.ID))
	}

	return res, nil
}
