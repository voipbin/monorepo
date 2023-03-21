package conversationhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

// Setup sets up the conversation webhook.
func (h *conversationHandler) Setup(ctx context.Context, customerID uuid.UUID, referenceType conversation.ReferenceType) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Setup",
		},
	)

	switch referenceType {
	case conversation.ReferenceTypeLine:
		if errSetup := h.lineHandler.Setup(ctx, customerID); errSetup != nil {
			log.Errorf("Could not setup the line. err: %v", errSetup)
			return errSetup
		}

	case conversation.ReferenceTypeMessage:
		log.Debugf("Nothing to do.")

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
		return fmt.Errorf("unsupported reference type")
	}

	return nil
}
