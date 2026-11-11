package eventhandler

import (
	"context"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"

	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/models/notification"
)

// EventConversationMessageCreated handles the conversation-manager's conversation_message_created event.
// Only an incoming message of a conversation that an agent owns is notified: a conversation without an owner is
// handled by a flow or an AI, and an outgoing message is the agent's own.
func (h *eventHandler) EventConversationMessageCreated(ctx context.Context, m *cvmessage.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":            "EventConversationMessageCreated",
		"customer_id":     m.CustomerID,
		"conversation_id": m.ConversationID,
		"message_id":      m.ID,
	})

	if m.Direction != cvmessage.DirectionIncoming || m.CustomerID == uuid.Nil || m.ConversationID == uuid.Nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, h.eventTimeout)
	defer cancel()

	cv, err := h.reqHandler.ConversationV1ConversationGet(ctx, m.ConversationID)
	if err != nil {
		log.Errorf("Could not get the conversation. err: %v", err)
		return fmt.Errorf("could not get the conversation. err: %w", err)
	}

	// the lookup has no customer scope, so the customer is checked here
	if cv == nil || cv.CustomerID != m.CustomerID {
		log.Warn("The conversation does not belong to the customer of the message. Skipping.")
		return nil
	}
	if cv.OwnerType != commonidentity.OwnerTypeAgent || cv.OwnerID == uuid.Nil {
		return nil
	}

	targets, err := h.settingsHandler.FilterRecipients(ctx, m.CustomerID, []uuid.UUID{cv.OwnerID}, notification.KindConversationMessage, uuid.Nil)
	if err != nil {
		log.Errorf("Could not apply the settings. err: %v", err)
		return fmt.Errorf("could not apply the settings. err: %w", err)
	}

	h.sendAll(ctx, m.CustomerID, targets, notification.KindConversationMessage, map[string]string{
		"conversation_id": m.ConversationID.String(),
		"message_id":      m.ID.String(),
	})

	return nil
}
