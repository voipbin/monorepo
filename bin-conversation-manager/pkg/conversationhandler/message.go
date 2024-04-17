package conversationhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

// MessageSend sends the message
func (h *conversationHandler) MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []media.Media) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "MessageSend",
		"conversation_id": conversationID,
		"text":            text,
		"medias":          medias,
	})
	log.Debugf("MessageSend detail. conversation_id: %s", conversationID)

	// get conversation
	cv, err := h.Get(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation. err: %v", err)
		return nil, err
	}

	// send to conversation
	m, err := h.messageHandler.Send(ctx, cv, text, medias)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, err
	}

	return m, nil
}
