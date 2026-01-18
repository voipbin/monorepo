package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	talkmessage "monorepo/bin-talk-manager/models/message"
	talkparticipant "monorepo/bin-talk-manager/models/participant"
	tkchat "monorepo/bin-talk-manager/models/chat"
)

// extractResource extracts the resource name from event type (e.g., "message" from "message_created")
func (h *subscribeHandler) extractResource(eventType string) string {
	parts := strings.Split(eventType, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return eventType
}

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
	case tkchat.EventTypeChatCreated, tkchat.EventTypeChatDeleted:
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

	// Create topics for message creator
	topics := []string{}

	// Extract resource from event type (e.g., "message" from "message_created")
	resource := h.extractResource(m.Type)

	// Creator's topic (OLD FORMAT: agent_id:owner_id:resource:id)
	if msg.OwnerID != uuid.Nil {
		topics = append(topics,
			fmt.Sprintf("agent_id:%s:%s:%s", msg.OwnerID, resource, msg.ID))
	}

	// Note: customer_id topic is NOT published for talk events

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

				// Add topic for each other participant (OLD FORMAT)
				topics = append(topics,
					fmt.Sprintf("agent_id:%s:%s:%s", p.OwnerID, resource, msg.ID))
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
	talk := &tkchat.WebhookMessage{}
	if err := json.Unmarshal(m.Data, talk); err != nil {
		log.Errorf("Could not unmarshal talk: %v", err)
		return err
	}

	// Create topics
	topics := []string{}

	// Extract resource from event type (e.g., "chat" from "chat_created")
	resource := h.extractResource(m.Type)

	// Publish to all participants in the talk (OLD FORMAT: agent_id:owner_id:resource:id)
	for _, p := range talk.Participants {
		if p.OwnerID != uuid.Nil {
			topics = append(topics,
				fmt.Sprintf("agent_id:%s:%s:%s", p.OwnerID, resource, talk.ID))
		}
	}

	// Note: customer_id topic is NOT published for talk events

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

	// Extract resource from event type (e.g., "participant" from "participant_added")
	resource := h.extractResource(m.Type)

	// Participant's topic (OLD FORMAT: agent_id:owner_id:resource:id)
	if participant.OwnerID != uuid.Nil {
		topics = append(topics,
			fmt.Sprintf("agent_id:%s:%s:%s", participant.OwnerID, resource, participant.ID))
	}

	// Note: customer_id topic is NOT published for talk events

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
	participants, err := h.reqHandler.TalkV1ParticipantList(ctx, talkID)
	if err != nil {
		return nil, err
	}
	return participants, nil
}
