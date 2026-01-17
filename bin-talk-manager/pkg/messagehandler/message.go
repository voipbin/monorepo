package messagehandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
)

// MessageCreate creates a new message with threading validation
func (h *messageHandler) MessageCreate(ctx context.Context, req MessageCreateRequest) (*message.Message, error) {
	log.WithFields(log.Fields{
		"func":        "MessageCreate",
		"customer_id": req.CustomerID,
		"chat_id":     req.ChatID,
		"owner_id":    req.OwnerID,
	}).Debug("Creating message")

	// Validate required fields
	if req.CustomerID == uuid.Nil {
		return nil, errors.New("customer_id is required")
	}
	if req.ChatID == uuid.Nil {
		return nil, errors.New("chat_id is required")
	}
	if req.OwnerType == "" {
		return nil, errors.New("owner_type is required")
	}
	if req.OwnerID == uuid.Nil {
		return nil, errors.New("owner_id is required")
	}
	if req.Type == "" {
		return nil, errors.New("type is required")
	}
	if req.Type != message.TypeNormal && req.Type != message.TypeSystem {
		return nil, errors.New("type must be either 'normal' or 'system'")
	}

	// Validate Medias JSON format if provided
	if req.Medias != "" {
		var medias []message.Media
		if err := json.Unmarshal([]byte(req.Medias), &medias); err != nil {
			return nil, errors.Wrap(err, "invalid medias JSON format")
		}
	}

	// Validate talk exists
	talk, err := h.dbHandler.TalkGet(ctx, req.ChatID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get talk")
	}
	if talk == nil {
		return nil, errors.New("talk not found")
	}

	// Validate sender is a participant
	// Note: Participants don't have soft delete, so no need to check deleted flag
	participants, err := h.dbHandler.ParticipantList(ctx, map[participant.Field]any{
		participant.FieldChatID:  req.ChatID,
		participant.FieldOwnerID: req.OwnerID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to check participant")
	}
	if len(participants) == 0 {
		return nil, errors.New("sender is not a participant in this talk")
	}

	// Validate parent message if provided (threading validation)
	var parentID *uuid.UUID
	if req.ParentID != nil {
		parent, err := h.dbHandler.MessageGet(ctx, *req.ParentID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get parent message")
		}
		if parent == nil {
			return nil, errors.New("parent message not found")
		}

		// CRITICAL: Validate parent is in same talk (prevents cross-talk threading)
		if parent.ChatID != req.ChatID {
			return nil, errors.New("parent message must be in the same talk")
		}

		// INTENTIONALLY ALLOWED: Parent can be soft-deleted
		// Reason: Preserve thread structure even when parent messages are deleted
		// UI should display deleted parent as placeholder (e.g., "Message deleted")
		if parent.TMDelete != "" {
			log.WithFields(log.Fields{
				"parent_id": parent.ID,
				"chat_id":   req.ChatID,
			}).Debug("Creating reply to soft-deleted parent message (allowed)")
		}

		parentID = req.ParentID
	}

	// Initialize metadata with empty reactions array
	defaultMetadata := map[string]interface{}{
		"reactions": []interface{}{},
	}
	metadataJSON, err := json.Marshal(defaultMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal metadata")
	}

	// Create message
	msg := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: req.CustomerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerType(req.OwnerType),
			OwnerID:   req.OwnerID,
		},
		ChatID:   req.ChatID,
		ParentID: parentID,
		Type:     message.Type(req.Type),
		Text:     req.Text,
		Medias:   req.Medias,
		Metadata: string(metadataJSON),
	}

	if err := h.dbHandler.MessageCreate(ctx, msg); err != nil {
		return nil, errors.Wrap(err, "failed to create message")
	}

	log.WithFields(log.Fields{
		"func":       "MessageCreate",
		"message_id": msg.ID,
		"chat_id":    msg.ChatID,
	}).Debug("Message created successfully")

	// Publish webhook event
	h.publishMessageCreatedEvent(ctx, msg)

	return msg, nil
}

// MessageGet retrieves a message by ID
func (h *messageHandler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	return h.dbHandler.MessageGet(ctx, id)
}

// MessageList retrieves messages with filters and pagination
func (h *messageHandler) MessageList(ctx context.Context, filters map[message.Field]any, token string, size uint64) ([]*message.Message, error) {
	return h.dbHandler.MessageList(ctx, filters, token, size)
}

// MessageDelete soft-deletes a message
func (h *messageHandler) MessageDelete(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	// Get message first
	msg, err := h.dbHandler.MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get message")
	}
	if msg == nil {
		return nil, errors.New("message not found")
	}

	// Check if already deleted
	if msg.TMDelete != "" {
		return nil, errors.New("message already deleted")
	}

	// Soft delete (dbHandler sets tm_delete internally)
	if err := h.dbHandler.MessageDelete(ctx, id); err != nil {
		return nil, errors.Wrap(err, "failed to delete message")
	}

	// Get updated message with tm_delete set
	updatedMsg, err := h.dbHandler.MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get updated message")
	}

	// Publish webhook event
	h.publishMessageDeletedEvent(ctx, updatedMsg)

	return updatedMsg, nil
}

// publishMessageCreatedEvent publishes a webhook event for message creation
func (h *messageHandler) publishMessageCreatedEvent(ctx context.Context, msg *message.Message) {
	h.notifyHandler.PublishWebhookEvent(ctx, msg.CustomerID, message.EventTypeMessageCreated, msg)
}

// publishMessageDeletedEvent publishes a webhook event for message deletion
func (h *messageHandler) publishMessageDeletedEvent(ctx context.Context, msg *message.Message) {
	h.notifyHandler.PublishWebhookEvent(ctx, msg.CustomerID, message.EventTypeMessageDeleted, msg)
}
