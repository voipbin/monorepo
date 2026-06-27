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

	case conversation.TypeWhatsApp:
		return h.sendWhatsApp(ctx, cv, text)

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
	messageID := h.utilHandler.UUIDCreate()
	source, destination := DeriveEndpoints(cv, message.DirectionOutgoing)
	res, err := h.Create(ctx, MessageCreateArgs{
		ID:             messageID,
		CustomerID:     cv.CustomerID,
		ConversationID: cv.ID,
		Direction:      message.DirectionOutgoing,
		Status:         message.StatusProgressing,
		ReferenceType:  message.ReferenceTypeMessage,
		ReferenceID:    messageID,
		Text:           text,
		Medias:         medias,
		Source:         source,
		Destination:    destination,
	})
	if err != nil {
		log.Errorf("Could not create a message. err: %v", err)
		return nil, err
	}

	if errSend := h.smsHandler.Send(ctx, cv, messageID, text); errSend != nil {
		log.Errorf("Could not send the message. err: %v", errSend)
		_, _ = h.UpdateStatus(ctx, res.ID, message.StatusFailed)
		return nil, errSend
	}

	return res, nil
}

// sendWhatsApp sends a message via the WhatsApp Business Cloud API.
func (h *messageHandler) sendWhatsApp(ctx context.Context, cv *conversation.Conversation, text string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "sendWhatsApp",
		"conversation_id": cv.ID,
	})

	ac, err := h.accountHandler.Get(ctx, cv.AccountID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get account")
	}

	source, destination := DeriveEndpoints(cv, message.DirectionOutgoing)
	tmp, err := h.Create(ctx, MessageCreateArgs{
		ID:             uuid.Nil,
		CustomerID:     cv.CustomerID,
		ConversationID: cv.ID,
		Direction:      message.DirectionOutgoing,
		Status:         message.StatusProgressing,
		ReferenceType:  message.ReferenceTypeWhatsApp,
		ReferenceID:    uuid.Nil,
		Text:           text,
		Source:         source,
		Destination:    destination,
	})
	if err != nil {
		log.Errorf("Could not create message. err: %v", err)
		return nil, err
	}

	wamid, err := h.whatsappHandler.Send(ctx, cv, ac, text)
	if err != nil {
		log.Errorf("Could not send WhatsApp message. err: %v", err)
		_, _ = h.UpdateStatus(ctx, tmp.ID, message.StatusFailed)
		return nil, err
	}

	res, err := h.UpdateStatus(ctx, tmp.ID, message.StatusDone)
	if err != nil {
		log.Errorf("Could not update message status. err: %v", err)
		return nil, err
	}

	// Persist wamid for deduplication and status correlation.
	if errUpd := h.db.MessageUpdate(ctx, res.ID, map[message.Field]any{
		message.FieldTransactionID: wamid,
	}); errUpd != nil {
		log.Warnf("Could not persist wamid. message_id: %s, wamid: %s, err: %v", res.ID, wamid, errUpd)
	}

	log.Debugf("Sent WhatsApp message. wamid: %s", wamid)
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

	source, destination := DeriveEndpoints(cv, message.DirectionOutgoing)
	tmp, err := h.Create(ctx, MessageCreateArgs{
		ID:             uuid.Nil,
		CustomerID:     cv.CustomerID,
		ConversationID: cv.ID,
		Direction:      message.DirectionOutgoing,
		Status:         message.StatusProgressing,
		ReferenceType:  message.ReferenceTypeLine,
		ReferenceID:    uuid.Nil,
		Text:           text,
		Medias:         medias,
		Source:         source,
		Destination:    destination,
	})
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
