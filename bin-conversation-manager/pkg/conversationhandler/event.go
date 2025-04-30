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

	// var self commonaddress.Address
	// var peer commonaddress.Address
	// var direction message.Direction

	// for _, target := range mm.Targets {
	// 	if mm.Direction == mmmessage.DirectionInbound {
	// 		self = target.Destination
	// 		peer = *mm.Source
	// 		direction = message.DirectionIncoming
	// 	} else {
	// 		self = *mm.Source
	// 		peer = target.Destination
	// 		direction = message.DirectionOutgoing
	// 	}

	// 	// get conversation
	// 	cv, err := h.GetBySelfAndPeer(ctx, self, peer)
	// 	if err != nil {
	// 		log.WithFields(logrus.Fields{
	// 			"self": self,
	// 			"peer": peer,
	// 		}).Debugf("Could not find conversation. Create a new conversation. err: %v", err)

	// 		// create a new conversation
	// 		cv, err = h.Create(
	// 			ctx,
	// 			mm.CustomerID,
	// 			"conversation",
	// 			"conversation with "+peer.TargetName,
	// 			conversation.TypeMessage,
	// 			"", // because it's sms conversation, there is no dialog id
	// 			self,
	// 			peer,
	// 		)
	// 		if err != nil {
	// 			return errors.Wrapf(err, "Could not create a new conversation")
	// 		}
	// 		log.WithField("conversation", cv).Debugf("Created a new conversation. conversation_id: %s", cv.ID)
	// 	}
	// 	log.WithField("conversation", cv).Debugf("Found conversation. conversation_id: %s", cv.ID)

	// 	tmp, err := h.messageHandler.Get(ctx, mm.ID)
	// 	if err != nil {
	// 		log.Debugf("Could not find the message. Create a new message.")

	// 		m, err := h.messageHandler.Create(
	// 			ctx,
	// 			mm.ID,
	// 			cv.CustomerID,
	// 			cv.ID,
	// 			direction,
	// 			message.StatusDone,
	// 			message.ReferenceTypeMessage,
	// 			mm.ID,
	// 			"",
	// 			mm.Text,
	// 			[]media.Media{},
	// 		)
	// 		if err != nil {
	// 			return errors.Wrapf(err, "Could not create a message")
	// 		}
	// 		log.WithField("message", m).Debugf("Create a message. message_id: %s", m.ID)
	// 	} else {
	// 		log.WithField("message", tmp).Debugf("Found message. Updating the message status. message_id: %s", tmp.ID)
	// 		updated, err := h.messageHandler.UpdateStatus(ctx, tmp.ID, message.StatusDone)
	// 		if err != nil {
	// 			return errors.Wrapf(err, "Could not update the message")
	// 		}

	// 		log.WithField("message", updated).Debugf("Updated message. message_id: %s", updated.ID)
	// 	}
	// }

	// return nil
}
