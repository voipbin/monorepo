package messagehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

// Send sends the message to the given conversation
func (h *messageHandler) Send(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SendToConversation",
		"conversation": cv,
		"text":         text,
		"medias":       medias,
	})
	log.Debugf("Sending a message to the conversation. conversation_id: %s, reference_type: %s", cv.ID, cv.Type)

	switch cv.Type {
	case conversation.TypeLine:
		return h.sendLine(ctx, cv, text, medias)

	case conversation.TypeMessage:
		return h.sendSMS(ctx, cv, text, medias)

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", cv.Type)
		return nil, fmt.Errorf("unsupported reference type. reference_type: %s", cv.Type)
	}
}

// sendSMS sends the message to the sms type of conversation.
func (h *messageHandler) sendSMS(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "sendSMS",
		"conversation": cv,
		"text":         text,
		"medias":       medias,
	})

	// create a sent message
	transactionID := uuid.Must(uuid.NewV4()).String()
	tmp, err := h.Create(ctx, cv.CustomerID, cv.ID, message.DirectionOutgoing, message.StatusProgressing, cv.Type, cv.ReferenceID, transactionID, text, medias)
	if err != nil {
		log.Errorf("Could not create a message. err: %v", err)
		return nil, err
	}

	if errSend := h.smsHandler.Send(ctx, cv, transactionID, text); errSend != nil {
		log.Errorf("Could not send the message. err: %v", errSend)
		_, _ = h.UpdateStatus(ctx, tmp.ID, message.StatusFailed)
		return nil, errSend
	}

	res, err := h.UpdateStatus(ctx, tmp.ID, message.StatusDone)
	if err != nil {
		log.Errorf("Could not update the message status. err: %v", err)
		return nil, err
	}

	return res, nil
}

// sendLine sends the message to the line type of conversation.
func (h *messageHandler) sendLine(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "sendLine",
		"conversation": cv,
		"text":         text,
		"medias":       medias,
	})

	// get account
	ac, err := h.accountHandler.Get(ctx, cv.AccountID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return nil, errors.Wrap(err, "could not get account")
	}

	// create a sent message
	tmp, err := h.Create(ctx, cv.CustomerID, cv.ID, message.DirectionOutgoing, message.StatusProgressing, cv.Type, cv.ReferenceID, "", text, medias)
	if err != nil {
		log.Errorf("Could not create a message. err: %v", err)
		return nil, err
	}

	if errSend := h.lineHandler.Send(ctx, cv, ac, text, medias); errSend != nil {
		log.Errorf("Could not send the message. err: %v", errSend)
		_, _ = h.UpdateStatus(ctx, tmp.ID, message.StatusFailed)
		return nil, errSend
	}

	res, err := h.UpdateStatus(ctx, tmp.ID, message.StatusDone)
	if err != nil {
		log.Errorf("Could not update the message status. err: %v", err)
		return nil, err
	}

	return res, nil
}
