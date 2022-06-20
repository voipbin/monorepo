package linehandler

import (
	"context"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
)

// Send sends the message to the destination
func (h *lineHandler) Send(ctx context.Context, customerID uuid.UUID, destination string, text string, medias []media.Media) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Send",
		},
	)
	log.Debug("Sending a message.")

	// get clinet
	c, err := h.getClient(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get client. err: %v", err)
		return err
	}

	var messages []linebot.SendingMessage

	if text != "" {
		messages = append(messages, linebot.NewTextMessage(text))
	}

	for _, tmp := range medias {
		log.Debugf("We've got media send request, but it's not support yet. media_type: %s", tmp.Type)
	}

	for _, tmp := range messages {
		// send a message to the destination
		res, err := c.PushMessage(destination, tmp).Do()
		if err != nil {
			log.Errorf("Could not send the message. err: %v", err)
			return err
		}
		log.WithField("message", res).Debugf("Sent a message request. request_id: %s", res.RequestID)
	}

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

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}

	res, err := linebot.New(
		a.LineSecret,
		a.LineToken,
		linebot.WithHTTPClient(client),
	)
	if err != nil {
		log.Errorf("Could not initiate linebot. err: %v", err)
		return nil, err
	}

	return res, nil
}
