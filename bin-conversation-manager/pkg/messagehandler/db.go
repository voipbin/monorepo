package messagehandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

// Create creates a message and returns a created message.
func (h *messageHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	conversationID uuid.UUID,
	direction message.Direction,
	status message.Status,
	referenceType message.ReferenceType,
	referenceID uuid.UUID,
	transactionID string,
	text string,
	medias []media.Media,
) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Create",
		"id":              id,
		"conversation_id": conversationID,
		"reference_type":  referenceType,
		"reference_id":    referenceID,
		"transaction_id":  transactionID,
	})
	log.Debugf("Creating a new message. reference_type: %s, reference_id: %s", referenceType, referenceID)

	// create a message
	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
		log = log.WithField("id", id)
		log.Debugf("The given id is nil. Created a new id. message_id: %s", id)
	}

	m := &message.Message{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ConversationID: conversationID,
		Direction:      direction,
		Status:         status,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		TransactionID: transactionID,

		Text:   text,
		Medias: medias,
	}

	if errCreate := h.db.MessageCreate(ctx, m); errCreate != nil {
		return nil, errors.Wrapf(errCreate, "Could not create a message.")
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get created message.")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageCreated, res)

	return res, nil
}

// Delete deletes the message and return the deleted message
func (h *messageHandler) Delete(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	if err := h.db.MessageDelete(ctx, id); err != nil {
		return nil, errors.Wrapf(err, "Could not delete the message.")
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get deleted message.")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageDeleted, res)

	return res, nil
}

// Gets returns list of messages of the given filters
func (h *messageHandler) Gets(ctx context.Context, pageToken string, pageSize uint64, filters map[message.Field]any) ([]*message.Message, error) {
	res, err := h.db.MessageGets(ctx, pageToken, pageSize, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get messages.")
	}

	return res, nil
}

// UpdateStatus returns list of messages of the given conversation
func (h *messageHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status message.Status) (*message.Message, error) {
	fields := map[message.Field]any{
		message.FieldStatus: status,
	}

	if errUpdate := h.db.MessageUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "Could not update the message.")
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get updated message.")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageUpdated, res)

	return res, nil
}

// Get returns message by id
func (h *messageHandler) Get(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get message.")
	}

	return res, nil
}
