package chathandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-talk-manager/models/chat"
)

// ChatCreate creates a new talk
func (h *chatHandler) ChatCreate(ctx context.Context, customerID uuid.UUID, chatType chat.Type) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatCreate",
		"customer_id": customerID,
		"type":        chatType,
	})
	log.Debug("Creating a new talk")

	// Validate input
	if customerID == uuid.Nil {
		log.Error("Invalid customer ID: nil UUID")
		return nil, errors.New("customer ID cannot be nil")
	}

	// Validate chat type
	if chatType != chat.TypeNormal && chatType != chat.TypeGroup {
		log.Errorf("Invalid chat type: %s", chatType)
		return nil, errors.Errorf("invalid chat type: %s", chatType)
	}

	// Create chat object
	t := &chat.Chat{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: customerID,
		},
		Type: chatType,
	}

	// Save to database
	err := h.dbHandler.ChatCreate(ctx, t)
	if err != nil {
		log.Errorf("Failed to create chat in database. err: %v", err)
		return nil, errors.Wrap(err, "failed to create chat in database")
	}

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, t.CustomerID, chat.EventTypeChatCreated, t)

	log.WithField("chat_id", t.ID).Debug("Chat created successfully")
	return t, nil
}

// ChatGet retrieves a chat by ID
func (h *chatHandler) ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ChatGet",
		"chat_id": id,
	})

	t, err := h.dbHandler.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Failed to get talk. err: %v", err)
		return nil, errors.Wrap(err, "failed to get talk")
	}

	return t, nil
}

// ChatList retrieves talks with filters and pagination
func (h *chatHandler) ChatList(ctx context.Context, filters map[chat.Field]any, token string, size uint64) ([]*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ChatList",
		"filters": filters,
		"token":   token,
		"size":    size,
	})

	talks, err := h.dbHandler.ChatList(ctx, filters, token, size)
	if err != nil {
		log.Errorf("Failed to list talks. err: %v", err)
		return nil, errors.Wrap(err, "failed to list talks")
	}

	return talks, nil
}

// ChatDelete soft deletes a talk
func (h *chatHandler) ChatDelete(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ChatDelete",
		"chat_id": id,
	})
	log.Debug("Deleting talk")

	// Get chat before deletion for webhook
	t, err := h.dbHandler.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Failed to get chat before deletion. err: %v", err)
		return nil, errors.Wrap(err, "failed to get chat before deletion")
	}

	// Soft delete in database
	err = h.dbHandler.ChatDelete(ctx, id)
	if err != nil {
		log.Errorf("Failed to delete talk. err: %v", err)
		return nil, errors.Wrap(err, "failed to delete talk")
	}

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, t.CustomerID, chat.EventTypeChatDeleted, t)

	log.Debug("Chat deleted successfully")
	return t, nil
}
