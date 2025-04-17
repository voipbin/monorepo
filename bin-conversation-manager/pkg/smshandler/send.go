package smshandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
)

// Send sends the message to the destination
func (h *smsHandler) Send(ctx context.Context, cv *conversation.Conversation, messageID uuid.UUID, text string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Send",
		"conversation": cv,
		"message_id":   messageID,
		"text":         text,
	})
	log.Debug("Sending an sms.")

	destinations := []commonaddress.Address{
		cv.Peer,
	}

	// send
	tmp, err := h.reqHandler.MessageV1MessageSend(ctx, messageID, cv.CustomerID, &cv.Self, destinations, text)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return err
	}

	log.WithField("sms", tmp).Debugf("Sent a sms correctly. message_id: %s", tmp.ID)
	return nil
}
