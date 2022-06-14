package conversationhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
)

// MessageSend sends the message
func (h *conversationHandler) MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "MessageSend",
		},
	)
	log.Debugf("MessageSend detail. conversation_id: %s", conversationID)

	// get conversation
	cv, err := h.Get(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation. err: %v", err)
		return nil, err
	}

	m, err := h.messageHandler.SendToConversation(ctx, cv, text, medias)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, err
	}

	return m, nil
}

// // messageCreate creates a message and returns a created message.
// func (h *conversationHandler) messageCreate(
// 	ctx context.Context,
// 	customerID uuid.UUID,
// 	conversationID uuid.UUID,
// 	status message.Status,
// 	referenceType conversation.ReferenceType,
// 	referenceID string,
// 	sourceID string,
// 	messageType message.Type,
// 	messageData []byte,
// ) (*message.Message, error) {
// 	log := logrus.WithFields(
// 		logrus.Fields{
// 			"func": "messageCreate",
// 		},
// 	)
// 	log.Debugf("Creating a new message. reference_type: %s, reference_id: %s, source_id: %s", referenceType, referenceID, sourceID)

// 	// create a message
// 	m := &message.Message{
// 		ID:             uuid.Must(uuid.NewV4()),
// 		CustomerID:     customerID,
// 		ConversationID: conversationID,
// 		Status:         status,
// 		ReferenceType:  referenceType,
// 		ReferenceID:    referenceID,
// 		SourceID:       sourceID,

// 		Type: messageType,
// 		Data: messageData,

// 		TMCreate: dbhandler.GetCurTime(),
// 		TMUpdate: dbhandler.GetCurTime(),
// 		TMDelete: dbhandler.DefaultTimeStamp,
// 	}

// 	if errCreate := h.db.MessageCreate(ctx, m); errCreate != nil {
// 		log.Errorf("Could not create a message. err: %v", errCreate)
// 		return nil, errCreate
// 	}

// 	res, err := h.db.MessageGet(ctx, m.ID)
// 	if err != nil {
// 		log.Errorf("Could not get created message. err: %v", err)
// 		return nil, err
// 	}
// 	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageCreated, res)

// 	return res, nil
// }

// // MessageSendByReferenceInfo sends the message
// func (h *conversationHandler) MessageSendByReferenceInfo(ctx context.Context, customerID uuid.UUID, referenceType conversation.ReferenceType, referenceID string, messageType message.Type, messageData []byte) (*message.Message, error) {
// 	log := logrus.WithFields(
// 		logrus.Fields{
// 			"func": "MessageSend",
// 		},
// 	)
// 	log.Debugf("MessageSend detail. reference_type: %s, reference_id: %s", referenceType, referenceID)

// 	// get conversation
// 	var cv *conversation.Conversation
// 	var err error
// 	cv, err = h.GetByReferenceInfo(ctx, referenceType, referenceID)
// 	if err != nil {

// 		// create a new one
// 		if referenceType == conversation.ReferenceTypeMessage {
// 			cv, err = h.Create(ctx, customerID, "conversation", "conversation detail", referenceType, referenceID, []participant.Participant{})
// 			if err != nil {
// 				log.Errorf("Could not create a new conversation. err: %v", err)
// 				return nil, err
// 			}
// 		} else {
// 			log.Errorf("Could not get conversation. err: %v", err)
// 			return nil, err
// 		}
// 	}

// 	m, err := h.messageSend(ctx, cv, messageType, messageData)
// 	if err != nil {
// 		log.Errorf("Could not send the message correctly. err: %v", err)
// 		return nil, err
// 	}

// 	return m, nil
// }

// // messageSend sends the message to the given conversation
// func (h *conversationHandler) messageSend(ctx context.Context, cv *conversation.Conversation, messageType message.Type, messageData []byte) (*message.Message, error) {
// 	log := logrus.WithFields(
// 		logrus.Fields{
// 			"func": "messageSend",
// 		},
// 	)
// 	log.Debugf("messageSend detail. conversation_id: %s", cv.ID)

// 	var err error
// 	switch cv.ReferenceType {
// 	case conversation.ReferenceTypeLine:
// 		err = h.lineHandler.Send(ctx, cv.CustomerID, cv.ReferenceID, messageType, messageData)

// 	default:
// 		log.Errorf("Unsupported reference type. reference_type: %s", cv.ReferenceType)
// 		err = fmt.Errorf("unsupported reference type. reference_type: %s", cv.ReferenceType)
// 	}

// 	if err != nil {
// 		log.Errorf("Could not send the data. err: %v", err)
// 		return nil, err
// 	}

// 	res, err := h.messageCreate(ctx, cv.CustomerID, cv.ID, message.StatusSent, cv.ReferenceType, cv.ReferenceID, "", messageType, messageData)
// 	if err != nil {
// 		log.Errorf("Could not create a message. err: %v", err)
// 		return nil, err
// 	}

// 	return res, nil
// }

// // MessageGetsByConversationID returns list of messages of the given conversation
// func (h *conversationHandler) MessageGetsByConversationID(ctx context.Context, conversationID uuid.UUID, pageToken string, pageSize uint64) ([]*message.Message, error) {
// 	log := logrus.WithFields(
// 		logrus.Fields{
// 			"func":            "MessageGetsByConversationID",
// 			"conversation_id": conversationID,
// 		},
// 	)
// 	log.Debugf("messageSend detail. conversation_id: %s", conversationID)

// 	res, err := h.db.MessageGetsByConversationID(ctx, conversationID, pageToken, pageSize)
// 	if err != nil {
// 		log.Errorf("Could not get messages. err: %v", err)
// 		return nil, err
// 	}

// 	return res, nil
// }
