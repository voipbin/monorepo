package conversationhandler

import (
	"context"
	"encoding/json"
	"fmt"

	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
)

// Event returns list of messages of the given conversation
func (h *conversationHandler) Event(ctx context.Context, referenceType conversation.Type, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Event",
		"reference_type": referenceType,
		"data":           data,
	})

	switch referenceType {
	case conversation.TypeMessage:
		if err := h.eventSMS(ctx, data); err != nil {
			log.Errorf("Could not handle the sms type event. err: %v", err)
			return err
		}

	default:
		log.Errorf("Could not find reference type handler. reference_type: %s", referenceType)
		return fmt.Errorf("reference type handler not found")
	}

	return nil
}

// eventSMS handle the sms type of hook message
func (h *conversationHandler) eventSMS(ctx context.Context, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "eventSMS",
		"data": data,
	})
	log.Debugf("Received message.")

	mm := mmmessage.Message{}
	if err := json.Unmarshal(data, &mm); err != nil {
		return errors.Wrapf(err, "Could not unmarshal the data")
	}

	switch mm.Direction {
	case mmmessage.DirectionInbound:
		if errEvent := h.MessageEventReceived(ctx, &mm); errEvent != nil {
			return errors.Wrapf(errEvent, "Could not handle the event correctly")
		}
		return nil

	case mmmessage.DirectionOutbound:
		if errEvent := h.MessageEventSent(ctx, &mm); errEvent != nil {
			return errors.Wrapf(errEvent, "Could not handle the event correctly")
		}
		return nil

	default:
		return errors.Errorf("could not find the direction. direction: %s", mm.Direction)
	}
}
