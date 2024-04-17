package linehandler

import (
	"context"
	"net/http"
	"time"

	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
)

// Send sends the message to the destination
func (h *lineHandler) Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string, medias []media.Media) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Send",
		"conversation": cv,
		"text":         text,
		"medias":       medias,
	})
	log.Debug("Sending a message.")

	// get clinet
	c, err := h.getClient(ctx, ac)
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
		res, err := c.PushMessage(cv.ReferenceID, tmp).Do()
		if err != nil {
			log.Errorf("Could not send the message. err: %v", err)
			return err
		}
		log.WithField("message", res).Debugf("Sent a message request. request_id: %s", res.RequestID)
	}

	return nil
}

// getClient returns given customer's line client.
func (h *lineHandler) getClient(ctx context.Context, ac *account.Account) (*linebot.Client, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "getClient",
		"account_id": ac.ID,
	})

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}

	res, err := linebot.New(
		ac.Secret,
		ac.Token,
		linebot.WithHTTPClient(client),
	)
	if err != nil {
		log.Errorf("Could not initiate linebot. err: %v", err)
		return nil, err
	}

	return res, nil
}
