package smshandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
)

// Send sends the message to the destination
func (h *smsHandler) Send(ctx context.Context, cv *conversation.Conversation, messageID uuid.UUID, text string, medias []media.Media) error {
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

	// send. A message without medias takes the sms path as before. With medias the id of each media
	// is the storage file id and the message goes to the mms route (message-manager validates the
	// files and their owner).
	var tmp *mmmessage.Message
	var err error
	if len(medias) == 0 {
		tmp, err = h.reqHandler.MessageV1MessageSend(ctx, messageID, cv.CustomerID, &cv.Self, destinations, text)
	} else {
		fileIDs := make([]uuid.UUID, len(medias))
		for i, m := range medias {
			fileIDs[i] = m.ID
		}
		log.Debugf("Sending an mms. media_count: %d", len(fileIDs))
		tmp, err = h.reqHandler.MessageV1MessageSendMMS(ctx, messageID, cv.CustomerID, &cv.Self, destinations, text, fileIDs)
	}
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return err
	}

	log.WithField("sms", tmp).Debugf("Sent a sms correctly. message_id: %s", tmp.ID)
	return nil
}
