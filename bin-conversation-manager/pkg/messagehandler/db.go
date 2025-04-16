package messagehandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

// Create creates a message and returns a created message.
func (h *messageHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	conversationID uuid.UUID,
	direction message.Direction,
	status message.Status,
	referenceType conversation.ReferenceType,
	referenceID string,
	transactionID string,
	// source *commonaddress.Address,
	// destination *commonaddress.Address,
	text string,
	medias []media.Media,
) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Create",
		"conversation_id": conversationID,
		"reference_type":  referenceType,
		"reference_id":    referenceID,
		"transaction_id":  transactionID,
	})
	log.Debugf("Creating a new message. reference_type: %s, reference_id: %s", referenceType, referenceID)

	// create a message
	id := h.utilHandler.UUIDCreate()
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
		log.Errorf("Could not create a message. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created message. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageCreated, res)

	return res, nil
}

// Delete deletes the message and return the deleted message
func (h *messageHandler) Delete(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"message_id": id,
	})

	if err := h.db.MessageDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the message. err: %v", err)
		return nil, err
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted message. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageDeleted, res)

	return res, nil
}

// GetsByConversationID returns list of messages of the given conversation
func (h *messageHandler) GetsByConversationID(ctx context.Context, conversationID uuid.UUID, pageToken string, pageSize uint64) ([]*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetsByConversationID",
		"conversation_id": conversationID,
	})

	res, err := h.db.MessageGetsByConversationID(ctx, conversationID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get messages. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByTransactionID returns list of messages of the given transaction id
func (h *messageHandler) GetsByTransactionID(ctx context.Context, transactionID string, pageToken string, pageSize uint64) ([]*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "GetsByConversationID",
		"transaction_id": transactionID,
	})

	res, err := h.db.MessageGetsByTransactionID(ctx, transactionID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get messages. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateStatus returns list of messages of the given conversation
func (h *messageHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status message.Status) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "UpdateStatus",
		"message_id": id,
	})

	if err := h.db.MessageUpdateStatus(ctx, id, status); err != nil {
		log.Errorf("Could not update the message status. err: %v", err)
		return nil, err
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated message. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageUpdated, res)

	return res, nil
}
