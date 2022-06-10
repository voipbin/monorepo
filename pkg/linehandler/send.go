package linehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
)

// Send sends the message to the destination
func (h *lineHandler) Send(ctx context.Context, customerID uuid.UUID, destination string, messageType message.Type, messageData []byte) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Send",
		},
	)
	log.Debug("Deleting the flow.")

	// get clinet
	c, err := h.getClient(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get client. err: %v", err)
		return err
	}

	var m linebot.SendingMessage
	switch messageType {
	case message.TypeText:
		m = linebot.NewTextMessage(string(messageData[:]))

	default:
		// currently, only text type supported.
		log.Errorf("Unsupported messate type. message_type: %s", messageType)
		return fmt.Errorf("unsupported message type. message_type: %s", messageType)
	}

	// send a message to the destination
	tmp, err := c.PushMessage(destination, m).Do()
	if err != nil {
		log.Errorf("Could not send the message. err: %v", err)
		return err
	}

	log.Debugf("Sent the message correctly. request_id: %s", tmp.RequestID)

	return nil
}

// getClient returns given customer's line client.
func (h *lineHandler) getClient(ctx context.Context, customerID uuid.UUID) (*linebot.Client, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "getClient",
		},
	)

	// get secret/token
	a, err := h.accountHandler.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return nil, err
	}

	res, err := linebot.New(a.LineSecret, a.LineToken)
	if err != nil {
		log.Errorf("Could not initiate linebot. err: %v", err)
		return nil, err
	}

	return res, nil
}
