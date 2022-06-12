package conversationhandler

import (
	"context"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
)

// Hook handles hooked event
func (h *conversationHandler) Hook(ctx context.Context, uri string, data []byte) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Hook",
		},
	)
	log.Debugf("Hook detail. uri: %s", uri)

	// "https://hook.voipbin.net/v1.0/conversation/customers/<customer_id>/line",
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	// /v1.0/conversation/customers/<customer_id>/line
	tmpVals := strings.Split(u.Path, "/")
	customerID := uuid.FromStringOrNil(tmpVals[4])
	referenceType := tmpVals[5]
	log.Debugf("Parsed data. customer_id: %s, provider: %s", customerID, referenceType)

	switch referenceType {
	case string(conversation.ReferenceTypeLine):
		// line message
		if errEvent := h.eventLine(ctx, customerID, data); errEvent != nil {
			log.Errorf("Could not handle the event type line. err: %v", errEvent)
			return errEvent
		}
	}

	return nil
}

// eventLine handle the line type of hook message
func (h *conversationHandler) eventLine(ctx context.Context, customerID uuid.UUID, data []byte) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "hookLine",
		},
	)

	// parse a messages
	conversations, messages, err := h.lineHandler.Event(ctx, customerID, data)
	if err != nil {
		log.Errorf("Could not parse the message. err: %v", err)
		return err
	}

	// conversations
	for _, tmp := range conversations {
		cv, err := h.Create(ctx, tmp.CustomerID, tmp.Name, tmp.Detail, tmp.ReferenceType, tmp.ReferenceID, tmp.Participants)
		if err != nil {
			log.Errorf("Could not create a new conversation. err: %v", err)
			break
		}
		log.Debugf("Created a new conversation. conversation_id: %s", cv.ID)
	}

	// messages
	for _, tmp := range messages {

		// get converstation
		cv, err := h.GetByReferenceInfo(ctx, tmp.ReferenceType, tmp.ReferenceID)
		if err != nil {
			log.Debugf("Could not find conversation. Create a new conversation.")

			// create a new conversation
			cv, err = h.Create(ctx, tmp.CustomerID, "conversation", "conversation detail", conversation.ReferenceTypeLine, tmp.ReferenceID, []commonaddress.Address{})
			if err != nil {
				log.Errorf("Could not create a new conversation. err: %v", err)
				continue
			}
		}

		// create a message
		m, err := h.messageHandler.Create(ctx, cv.CustomerID, cv.ID, message.StatusReceived, conversation.ReferenceTypeLine, tmp.ReferenceID, tmp.SourceTarget, tmp.Type, tmp.Data)
		if err != nil {
			log.Errorf("Could not create a message. err: %v", err)
			continue
		}
		log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	}

	return nil
}
