package messagehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
)

// SendToConversation sends the message to the given conversation
func (h *messageHandler) SendToConversation(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SendToConversation",
		"conversation": cv,
		"text":         text,
		"medias":       medias,
	})
	log.Debugf("Sending a message to the conversation. conversation_id: %s, reference_type: %s", cv.ID, cv.ReferenceType)

	switch cv.ReferenceType {
	case conversation.ReferenceTypeLine:
		return h.sendToConversationLine(ctx, cv, text, medias)

	case conversation.ReferenceTypeMessage:
		return h.sendToConversationSMS(ctx, cv, text, medias)

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", cv.ReferenceType)
		return nil, fmt.Errorf("unsupported reference type. reference_type: %s", cv.ReferenceType)
	}
}

// sendToConversationSMS sends the message to the sms type of conversation.
func (h *messageHandler) sendToConversationSMS(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "sendToConversationSMS",
		"conversation": cv,
		"text":         text,
		"medias":       medias,
	})

	// create a sent message
	transactionID := uuid.Must(uuid.NewV4()).String()
	tmp, err := h.Create(ctx, cv.CustomerID, cv.ID, message.DirectionOutgoing, message.StatusSending, cv.ReferenceType, cv.ReferenceID, transactionID, cv.Source, text, medias)
	if err != nil {
		log.Errorf("Could not create a message. err: %v", err)
		return nil, err
	}

	if errSend := h.smsHandler.Send(ctx, cv, transactionID, text); errSend != nil {
		log.Errorf("Could not send the message. err: %v", errSend)
		_, _ = h.UpdateStatus(ctx, tmp.ID, message.StatusFailed)
		return nil, errSend
	}

	res, err := h.UpdateStatus(ctx, tmp.ID, message.StatusSent)
	if err != nil {
		log.Errorf("Could not update the message status. err: %v", err)
		return nil, err
	}

	return res, nil
}

// sendToConversationLine sends the message to the line type of conversation.
func (h *messageHandler) sendToConversationLine(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "sendToConversationLine",
	})

	// get account
	ac, err := h.accountHandler.Get(ctx, cv.AccountID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return nil, errors.Wrap(err, "could not get account")
	}

	// create a sent message
	tmp, err := h.Create(ctx, cv.CustomerID, cv.ID, message.DirectionOutgoing, message.StatusSending, cv.ReferenceType, cv.ReferenceID, "", cv.Source, text, medias)
	if err != nil {
		log.Errorf("Could not create a message. err: %v", err)
		return nil, err
	}

	if errSend := h.lineHandler.Send(ctx, cv, ac, text, medias); errSend != nil {
		log.Errorf("Could not send the message. err: %v", errSend)
		_, _ = h.UpdateStatus(ctx, tmp.ID, message.StatusFailed)
		return nil, errSend
	}

	res, err := h.UpdateStatus(ctx, tmp.ID, message.StatusSent)
	if err != nil {
		log.Errorf("Could not update the message status. err: %v", err)
		return nil, err
	}

	return res, nil
}
