package messagehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/message"
)

// Get returns the message.
func (h *messageHandler) Get(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get message. err: %w", err)
	}

	return res, nil
}

// List returns messages.
func (h *messageHandler) List(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error) {
	res, err := h.db.MessageList(ctx, size, token, filters)
	if err != nil {
		return nil, fmt.Errorf("could not list messages. err: %w", err)
	}

	return res, nil
}

// Delete deletes the message.
func (h *messageHandler) Delete(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"message_id": id,
	})
	log.Debug("Deleting the message.")

	if err := h.db.MessageDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the message info. err: %v", err)
		return nil, err
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted message info. err: %v", err)
		return nil, err
	}

	return res, nil
}
