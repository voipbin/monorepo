package reactionhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-talk-manager/models/message"
)

// ReactionAdd adds a reaction to a message with idempotent behavior
func (h *reactionHandler) ReactionAdd(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ReactionAdd",
		"message_id": messageID,
		"emoji":      emoji,
		"owner_id":   ownerID,
	})
	log.Debug("Adding reaction")

	// Validate required fields
	if messageID == uuid.Nil {
		return nil, errors.New("message_id is required")
	}
	if emoji == "" {
		return nil, errors.New("emoji is required")
	}
	if ownerType == "" {
		return nil, errors.New("owner_type is required")
	}
	if ownerID == uuid.Nil {
		return nil, errors.New("owner_id is required")
	}

	// Check if reaction already exists (idempotent check)
	m, err := h.dbHandler.MessageGet(ctx, messageID)
	if err != nil {
		log.Errorf("Failed to get message: %v", err)
		return nil, errors.Wrap(err, "failed to get message")
	}
	if m == nil {
		return nil, errors.New("message not found")
	}

	for _, r := range m.Metadata.Reactions {
		if r.Emoji == emoji && r.OwnerType == ownerType && r.OwnerID == ownerID {
			// Already exists, return current message (idempotent)
			log.Debug("Reaction already exists, returning current message")
			h.publishReactionUpdated(ctx, m)
			return m, nil
		}
	}

	// Add reaction atomically using MySQL JSON functions
	// This prevents race conditions when multiple users add reactions simultaneously
	now := h.utilHandler.TimeGetCurTime()
	reaction := message.Reaction{
		Emoji:     emoji,
		OwnerType: ownerType,
		OwnerID:   ownerID,
		TMCreate:  now,
	}
	reactionJSON, err := json.Marshal(reaction)
	if err != nil {
		log.Errorf("Failed to marshal reaction: %v", err)
		return nil, errors.Wrap(err, "failed to marshal reaction")
	}

	err = h.dbHandler.MessageAddReactionAtomic(ctx, messageID, string(reactionJSON))
	if err != nil {
		log.Errorf("Failed to add reaction atomically: %v", err)
		return nil, errors.Wrap(err, "failed to add reaction")
	}

	// Refresh and publish
	m, err = h.dbHandler.MessageGet(ctx, messageID)
	if err != nil {
		log.Errorf("Failed to get updated message: %v", err)
		return nil, errors.Wrap(err, "failed to get updated message")
	}

	log.Debugf("Reaction added successfully. message_id: %s, emoji: %s", messageID, emoji)

	h.publishReactionUpdated(ctx, m)

	return m, nil
}

// ReactionRemove removes a reaction from a message
func (h *reactionHandler) ReactionRemove(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ReactionRemove",
		"message_id": messageID,
		"emoji":      emoji,
		"owner_id":   ownerID,
	})
	log.Debug("Removing reaction")

	// Validate required fields
	if messageID == uuid.Nil {
		return nil, errors.New("message_id is required")
	}
	if emoji == "" {
		return nil, errors.New("emoji is required")
	}
	if ownerType == "" {
		return nil, errors.New("owner_type is required")
	}
	if ownerID == uuid.Nil {
		return nil, errors.New("owner_id is required")
	}

	// Remove reaction atomically
	err := h.dbHandler.MessageRemoveReactionAtomic(ctx, messageID, emoji, ownerType, ownerID)
	if err != nil {
		log.Errorf("Failed to remove reaction atomically: %v", err)
		return nil, errors.Wrap(err, "failed to remove reaction")
	}

	// Refresh and publish
	m, err := h.dbHandler.MessageGet(ctx, messageID)
	if err != nil {
		log.Errorf("Failed to get updated message: %v", err)
		return nil, errors.Wrap(err, "failed to get updated message")
	}

	log.Debugf("Reaction removed successfully. message_id: %s, emoji: %s", messageID, emoji)

	h.publishReactionUpdated(ctx, m)

	return m, nil
}

// publishReactionUpdated publishes a single webhook event for both add and remove
func (h *reactionHandler) publishReactionUpdated(ctx context.Context, m *message.Message) {
	// Convert to WebhookMessage before publishing
	// This ensures medias is sent as []Media instead of JSON string
	wm, err := m.ConvertWebhookMessage()
	if err != nil {
		logrus.WithError(err).Error("Failed to convert message to webhook message")
		return
	}
	h.notifyHandler.PublishWebhookEvent(ctx, m.CustomerID, message.EventTypeMessageReactionUpdated, wm)
}
