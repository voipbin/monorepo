package messagehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
)

// Create creates a message and returns a created message.
func (h *messageHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	conversationID uuid.UUID,
	status message.Status,
	referenceType conversation.ReferenceType,
	referenceID string,
	sourceTarget string,
	messageType message.Type,
	messageData []byte,
) (*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "Create",
			"conversation_id": conversationID,
			"reference_type":  referenceType,
			"reference_id":    referenceID,
		},
	)
	log.Debugf("Creating a new message. reference_type: %s, reference_id: %s, source_target: %s", referenceType, referenceID, sourceTarget)

	// create a message
	m := &message.Message{
		ID:             uuid.Must(uuid.NewV4()),
		CustomerID:     customerID,
		ConversationID: conversationID,
		Status:         status,
		ReferenceType:  referenceType,
		ReferenceID:    referenceID,
		SourceTarget:   sourceTarget,

		Type: messageType,
		Data: messageData,

		TMCreate: dbhandler.GetCurTime(),
		TMUpdate: dbhandler.GetCurTime(),
		TMDelete: dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.MessageCreate(ctx, m); errCreate != nil {
		log.Errorf("Could not create a message. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.MessageGet(ctx, m.ID)
	if err != nil {
		log.Errorf("Could not get created message. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageCreated, res)

	return res, nil
}

// GetsByConversationID returns list of messages of the given conversation
func (h *messageHandler) GetsByConversationID(ctx context.Context, conversationID uuid.UUID, pageToken string, pageSize uint64) ([]*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "GetsByConversationID",
			"conversation_id": conversationID,
		},
	)

	res, err := h.db.MessageGetsByConversationID(ctx, conversationID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get messages. err: %v", err)
		return nil, err
	}

	return res, nil
}
