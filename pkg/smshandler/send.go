package smshandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

// Send sends the message to the destination
func (h *smsHandler) Send(ctx context.Context, cv *conversation.Conversation, transactionID string, text string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "Send",
			"conversation_id": cv.ID,
		},
	)
	log.Debug("Sending an sms.")

	destinations := []commonaddress.Address{
		{
			Target: cv.ReferenceID,
		},
	}

	id := uuid.FromStringOrNil(transactionID)

	// send
	tmp, err := h.reqHandler.MMV1MessageSend(ctx, id, cv.CustomerID, cv.Source, destinations, text)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return err
	}

	log.WithField("sms", tmp).Debugf("Sent a sms correctly. message_id: %s", tmp.ID)
	return nil
}
