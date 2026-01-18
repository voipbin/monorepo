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

// createTalkTopics generates both old and new format topics for talk events
// Note: customer_id topics are NOT generated for talk events (agent-only)
func (h *subscribeHandler) createTalkTopics(eventType string, ownerID uuid.UUID, resourceID uuid.UUID) []string {
	topics := []string{}

	if ownerID == uuid.Nil {
		return topics
	}

	// Extract resource from event type (e.g., "message" from "message_created")
	resource := h.extractResource(eventType)

	// OLD FORMAT (backward compatible): agent_id:OWNER_ID:resource:ID
	topics = append(topics, fmt.Sprintf("agent_id:%s:%s:%s", ownerID, resource, resourceID))

	// NEW FORMAT (service-namespaced): agent_id:OWNER_ID:talk:event_type:ID
	topics = append(topics, fmt.Sprintf("agent_id:%s:%s:%s:%s", ownerID, "talk", eventType, resourceID))

	return topics
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
	case tkchat.EventTypeChatCreated, tkchat.EventTypeChatDeleted, tkchat.EventTypeChatUpdated:
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

	// Create topics for message creator (both old and new formats)
	topics := h.createTalkTopics(m.Type, msg.OwnerID, msg.ID)

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

				// Add topics for each other participant (both formats)
				topics = append(topics, h.createTalkTopics(m.Type, p.OwnerID, msg.ID)...)
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

	// Create topics for all participants (both old and new formats)
	topics := []string{}
	for _, p := range talk.Participants {
		topics = append(topics, h.createTalkTopics(m.Type, p.OwnerID, talk.ID)...)
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

	// Create topics for the participant (both old and new formats)
	topics := h.createTalkTopics(m.Type, participant.OwnerID, participant.ID)

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
