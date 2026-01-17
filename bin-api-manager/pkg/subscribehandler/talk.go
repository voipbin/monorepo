package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	talkmessage "monorepo/bin-talk-manager/models/message"
	talkparticipant "monorepo/bin-talk-manager/models/participant"
	talktalk "monorepo/bin-talk-manager/models/talk"
)

// processEventTalkManager handles all events from talk-manager
func (h *subscribeHandler) processEventTalkManager(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "processEventTalkManager",
		"event":     m.Type,
		"publisher": m.Publisher,
	})
	log.Debugf("Processing talk-manager event")

	switch m.Type {
	case talkmessage.EventTypeMessageCreated, talkmessage.EventTypeMessageDeleted, talkmessage.EventTypeMessageReactionUpdated:
		return h.processEventTalkMessage(ctx, m)
	case talktalk.EventTypeTalkCreated, talktalk.EventTypeTalkDeleted:
		return h.processEventTalk(ctx, m)
	case talkparticipant.EventParticipantAdded, talkparticipant.EventParticipantRemoved:
		return h.processEventTalkParticipant(ctx, m)
	default:
		log.Warnf("Unknown talk-manager event type: %s", m.Type)
		return nil
	}
}

// processEventTalkMessage handles message events (created, deleted, reaction updated)
func (h *subscribeHandler) processEventTalkMessage(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventTalkMessage",
		"event": m.Type,
	})

	// Parse message webhook data
	msg := &talkmessage.WebhookMessage{}
	if err := json.Unmarshal(m.Data, msg); err != nil {
		log.Errorf("Could not unmarshal message: %v", err)
		return err
	}

	// Create topics for message creator and customer
	topics := []string{}
	service := h.getServiceNamespace(m.Publisher) // "talk"

	// Creator's topic
	if msg.OwnerID != uuid.Nil {
		topics = append(topics,
			fmt.Sprintf("agent_id:%s:%s:%s:%s", msg.OwnerID, service, m.Type, msg.ID))
	}

	// Customer topic
	if msg.CustomerID != uuid.Nil {
		topics = append(topics,
			fmt.Sprintf("customer_id:%s:%s:%s:%s", msg.CustomerID, service, m.Type, msg.ID))
	}

	// CRITICAL: Add topics for all talk participants (not just creator)
	// This ensures all participants in the talk receive the message notification
	if msg.ChatID != uuid.Nil {
		participants, err := h.getTalkParticipants(ctx, msg.ChatID)
		if err != nil {
			log.Warnf("Could not get talk participants: %v", err)
		} else {
			for _, p := range participants {
				// Skip message creator (already in topics)
				if p.OwnerID == msg.OwnerID {
					continue
				}

				// Add topic for each other participant
				topics = append(topics,
					fmt.Sprintf("agent_id:%s:%s:%s:%s", p.OwnerID, service, m.Type, msg.ID))
			}
		}
	}

	// Publish to all topics
	for _, topic := range topics {
		if err := h.zmqpubHandler.Publish(topic, string(m.Data)); err != nil {
			log.Errorf("Could not publish to topic %s: %v", topic, err)
			return err
		}
		log.Debugf("Published to topic: %s", topic)
	}

	return nil
}

// processEventTalk handles talk events (created, deleted)
func (h *subscribeHandler) processEventTalk(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventTalk",
		"event": m.Type,
	})

	// Parse talk webhook data
	talk := &talktalk.WebhookMessage{}
	if err := json.Unmarshal(m.Data, talk); err != nil {
		log.Errorf("Could not unmarshal talk: %v", err)
		return err
	}

	// Create topics
	topics := []string{}
	service := h.getServiceNamespace(m.Publisher) // "talk"

	// Customer topic
	if talk.CustomerID != uuid.Nil {
		topics = append(topics,
			fmt.Sprintf("customer_id:%s:%s:%s:%s", talk.CustomerID, service, m.Type, talk.ID))
	}

	// Publish to all topics
	for _, topic := range topics {
		if err := h.zmqpubHandler.Publish(topic, string(m.Data)); err != nil {
			log.Errorf("Could not publish to topic %s: %v", topic, err)
			return err
		}
		log.Debugf("Published to topic: %s", topic)
	}

	return nil
}

// processEventTalkParticipant handles participant events (added, removed)
func (h *subscribeHandler) processEventTalkParticipant(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventTalkParticipant",
		"event": m.Type,
	})

	// Parse participant webhook data
	participant := &talkparticipant.WebhookMessage{}
	if err := json.Unmarshal(m.Data, participant); err != nil {
		log.Errorf("Could not unmarshal participant: %v", err)
		return err
	}

	// Create topics
	topics := []string{}
	service := h.getServiceNamespace(m.Publisher) // "talk"

	// Participant's topic
	if participant.OwnerID != uuid.Nil {
		topics = append(topics,
			fmt.Sprintf("agent_id:%s:%s:%s:%s", participant.OwnerID, service, m.Type, participant.ID))
	}

	// Customer topic
	if participant.CustomerID != uuid.Nil {
		topics = append(topics,
			fmt.Sprintf("customer_id:%s:%s:%s:%s", participant.CustomerID, service, m.Type, participant.ID))
	}

	// Publish to all topics
	for _, topic := range topics {
		if err := h.zmqpubHandler.Publish(topic, string(m.Data)); err != nil {
			log.Errorf("Could not publish to topic %s: %v", topic, err)
			return err
		}
		log.Debugf("Published to topic: %s", topic)
	}

	return nil
}

// getTalkParticipants fetches all participants for a talk
func (h *subscribeHandler) getTalkParticipants(ctx context.Context, talkID uuid.UUID) ([]*talkparticipant.Participant, error) {
	// Call talk-manager via RPC to get participants
	// Use the requestHandler to make RPC call
	participants, err := h.reqHandler.TalkV1TalkParticipantList(ctx, talkID)
	if err != nil {
		return nil, err
	}
	return participants, nil
}
