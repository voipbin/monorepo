package conversationhandler

import (
	"context"
	"fmt"
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
	if len(tmpVals) < 6 {
		log.Debugf("Wrong hook request. Could not get customerID.")
		return fmt.Errorf("no customer info found")
	}

	customerID := uuid.FromStringOrNil(tmpVals[4])
	referenceType := tmpVals[5]
	log.Debugf("Parsed data. customer_id: %s, provider: %s", customerID, referenceType)

	switch referenceType {
	case string(conversation.ReferenceTypeLine):
		// line message
		if errEvent := h.hookLine(ctx, customerID, data); errEvent != nil {
			log.Errorf("Could not handle the event type line. err: %v", errEvent)
			return errEvent
		}

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
		return fmt.Errorf("unsupported reference type. reference_type: %s", referenceType)
	}

	return nil
}

// hookLine handle the line type of hook message
func (h *conversationHandler) hookLine(ctx context.Context, customerID uuid.UUID, data []byte) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "hookLine",
		},
	)

	// parse a messages
	conversations, messages, err := h.lineHandler.Hook(ctx, customerID, data)
	if err != nil {
		log.Errorf("Could not parse the message. err: %v", err)
		return err
	}

	// conversations
	for _, tmp := range conversations {
		cv, err := h.Create(ctx, tmp.CustomerID, tmp.Name, tmp.Detail, tmp.ReferenceType, tmp.ReferenceID, tmp.Source, tmp.Participants)
		if err != nil {
			log.Errorf("Could not create a new conversation. err: %v", err)
			break
		}
		log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
	}

	// messages
	for _, tmp := range messages {

		// get converstation
		cv, err := h.GetByReferenceInfo(ctx, tmp.ReferenceType, tmp.ReferenceID)
		if err != nil {
			log.Debugf("Could not find conversation. Create a new conversation.")

			// get address
			// get user info
			p, err := h.lineHandler.GetParticipant(ctx, customerID, tmp.ReferenceID)
			if err != nil {
				log.Errorf("Could not get participant info. err: %v", err)
				p = &commonaddress.Address{
					Type:       commonaddress.TypeLine,
					Target:     tmp.ReferenceID,
					TargetName: "Unknown",
				}
			}

			me := &commonaddress.Address{
				Type:       commonaddress.TypeLine,
				Target:     "",
				TargetName: "me",
			}

			// create a new conversation
			cv, err = h.Create(ctx, tmp.CustomerID, "conversation", "conversation detail", conversation.ReferenceTypeLine, tmp.ReferenceID, me, []commonaddress.Address{*me, *p})
			if err != nil {
				log.Errorf("Could not create a new conversation. err: %v", err)
				continue
			}
			log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
		}

		// create a message
		m, err := h.messageHandler.Create(ctx, cv.CustomerID, cv.ID, message.StatusReceived, conversation.ReferenceTypeLine, tmp.ReferenceID, "", tmp.Source, tmp.Text, tmp.Medias)
		if err != nil {
			log.Errorf("Could not create a message. err: %v", err)
			continue
		}
		log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	}

	return nil
}
