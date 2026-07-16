package subscribehandler

import (
	"context"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
)

// processEventWebchatMessageMessageCreated handles the webchat-manager's
// webchat_message_created event. Design doc §16 (message-manager
// pattern): mirrors processEventMessageMessageCreated exactly, but the
// event data itself is unmarshaled inside conversationHandler.eventWebchat
// (not here) since the payload shape differs from SMS's multi-target
// mmmessage.Message and doesn't need re-parsing at this layer.
func (h *subscribeHandler) processEventWebchatMessageMessageCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventWebchatMessageMessageCreated",
		"event": m,
	})

	if errEvent := h.conversationHandler.Event(ctx, conversation.TypeWebchat, m.Data); errEvent != nil {
		log.Errorf("Could not handle the event correctly. err: %v", errEvent)
		return errEvent
	}

	return nil
}
