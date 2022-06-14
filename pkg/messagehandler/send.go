package messagehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
)

// SendToConversation sends the message to the given conversation
func (h *messageHandler) SendToConversation(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "SendToConversation",
		},
	)
	log.Debugf("Sending a message to the conversation. conversation_id: %s, reference_type: %s", cv.ID, cv.ReferenceType)

	var err error
	switch cv.ReferenceType {
	case conversation.ReferenceTypeLine:
		err = h.lineHandler.Send(ctx, cv.CustomerID, cv.ReferenceID, text, medias)

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", cv.ReferenceType)
		err = fmt.Errorf("unsupported reference type. reference_type: %s", cv.ReferenceType)
	}

	if err != nil {
		log.Errorf("Could not send the data. err: %v", err)
		return nil, err
	}

	// create a sent message
	res, err := h.Create(ctx, cv.CustomerID, cv.ID, message.StatusSent, cv.ReferenceType, cv.ReferenceID, "", text, medias)
	if err != nil {
		log.Errorf("Could not create a message. err: %v", err)
		return nil, err
	}

	return res, nil
}
